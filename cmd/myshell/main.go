package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type (
	command     func([]string) (commandResult, commandStatus)
	commandType int
)

type (
	commandResult = string
	commandStatus = int
)

const (
	cmdBuiltin commandType = iota
	cmdSystem
	cmdNotFound
)

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")

		reader, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err == io.EOF {
			fmt.Fprintln(os.Stdout, "closing shell...")
			os.Exit(0)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "error: ", err.Error())
		}

		args := parseArgs(strings.TrimSpace(reader))
		if len(args) == 0 {
			continue
		}

		commandName := strings.TrimSpace(args[0])

		result, status := executeCommand(commandName, args)

		if status > 0 {
			fmt.Fprintln(os.Stderr, result)
		} else {
			fmt.Fprintln(os.Stdout, result)
		}
	}
}

const (
	doubleQuoted = iota
	doubleQuotedEscaped
	unquoted
	unquotedEscaped
	singleQuoted
)

func parseArgs(input string) []string {
	var args []string
	currWord := ""
	state := unquoted

	for i, x := range input {
		startNextWord := i == len(input)-1
		nextState := state
		toAdd := ""

		if state == unquoted {
			switch x {
			case '"':
				nextState = doubleQuoted
			case '\'':
				nextState = singleQuoted
			case '\\':
				nextState = unquotedEscaped
			case ' ':
				startNextWord = startNextWord || len(currWord) > 0
			default:
				toAdd = string(x)
			}
		}

		if state == unquotedEscaped {
			nextState = unquoted
			toAdd = string(x)
		}

		if state == singleQuoted {
			if x == '\'' {
				nextState = unquoted
			} else {
				toAdd = string(x)
			}
		}

		if state == doubleQuoted {
			switch x {
			case '\\':
				nextState = doubleQuotedEscaped
			case '"':
				nextState = unquoted
			default:
				toAdd = string(x)
			}
		}

		if state == doubleQuotedEscaped {
			switch string(x) {
			case "\"", "$", "`", "\\", "\n":
				toAdd = string(x)
			default:
				toAdd = "\\" + string(x)
			}

			nextState = doubleQuoted
		}

		state = nextState
		currWord += toAdd

		if startNextWord {
			args = append(args, strings.TrimSpace(currWord))
			currWord = ""
		}
	}

	return args
}

func executeCommand(commandName string, inputs []string) (commandResult, commandStatus) {
	if builtinCommand, cmdType := getBuiltinCommand(commandName); cmdType == cmdBuiltin {
		return builtinCommand(inputs)
	}

	if cmdPath, cmdType := getSystemCommand(commandName); cmdType == cmdSystem {
		var status commandStatus = 1
		var result commandResult
		args := inputs[1:]
		cmd := exec.Command(cmdPath, args...)

		if output, err := cmd.Output(); err != nil {
			result = err.Error()
		} else {
			result = string(output)
		}

		return result, status
	}

	return handleNotFound(inputs)
}

func getBuiltinCommand(commandName string) (command, commandType) {
	builtins := map[string]command{
		"cd":   handleCd,
		"echo": handleEcho,
		"exit": handleExit,
		"pwd":  handlePwd,
		"type": handleType,
	}
	cmdType := cmdNotFound
	result, ok := builtins[commandName]
	if ok {
		cmdType = cmdBuiltin
	}

	return result, cmdType
}

func getSystemCommand(commandName string) (string, commandType) {
	paths := strings.Split(os.Getenv("PATH"), ":")
	cmdType := cmdNotFound
	var path string

	for _, x := range paths {
		cmdPath := filepath.Join(x, commandName)
		fileInfo, err := os.Stat(filepath.Join(x, commandName))

		if err == nil && fileInfo.Mode().Perm()&0o100 != 0 {
			path = cmdPath
			cmdType = cmdSystem
			break
		}
	}

	return path, cmdType
}

func handleExit(args []string) (commandResult, commandStatus) {
	var result commandResult
	status := 0

	if len(args) > 2 {
		result = fmt.Sprintf("too many arguments")
		status = 1
	}

	if len(args) > 1 {
		arg := args[1]
		x, err := strconv.Atoi(arg)
		if err != nil {
			status = x
			result = err.Error()
		}
	}

	// TODO: understand why this doesn't raise compiler errors
	os.Exit(status)

	return result, status
}

func handleCd(args []string) (commandResult, commandStatus) {
	var status commandStatus = 1
	var result commandResult
	path := strings.TrimSpace(strings.Join(args[1:], ""))

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			result = err.Error()
		}

		path = homeDir
	} else if fileInfo, err := os.Stat(path); err != nil || !fileInfo.IsDir() {
		result = fmt.Sprintf("cd: %s: No such file or directory", path)
	}

	err := os.Chdir(path)
	if err != nil {
		result = fmt.Sprintf("error: %s", err.Error())
	}

	if len(result) == 0 {
		status = 0
	}

	return result, status
}

func handleEcho(args []string) (commandResult, commandStatus) {
	status := 0
	values := args[1:]
	result := strings.Join(values, " ")

	return result, commandStatus(status)
}

func handlePwd(args []string) (commandResult, commandStatus) {
	var status commandStatus = 1
	var result commandResult

	if len(args[1:]) > 2 {
		result = "too many arguments"

		return result, status
	}

	wd, err := os.Getwd()
	if err != nil {
		result = err.Error()
	} else {
		result = wd
		status = 0
	}

	return result, status
}

func handleType(args []string) (commandResult, commandStatus) {
	commands := args[1:]
	typeByCommand := make(map[string]commandType)
	var status commandStatus = 1
	var result commandResult

	for _, x := range commands {
		if _, cmdType := getBuiltinCommand(x); cmdType != cmdNotFound {
			typeByCommand[x] = cmdType
		} else if cmdPath, cmdType := getSystemCommand(x); cmdType != cmdNotFound {
			typeByCommand[cmdPath] = cmdType
		} else {
			typeByCommand[x] = cmdNotFound
		}
	}

	for cmd, cmdType := range typeByCommand {
		switch cmdType {
		case cmdBuiltin:
			result = fmt.Sprintf("%s is a shell builtin", cmd)
			status = 0
		case cmdSystem:
			result = fmt.Sprintf("%s is %s", filepath.Base(cmd), cmd)
			status = 0
		default:
			result = fmt.Sprintf("%s: not found", cmd)
		}
	}

	return result, status
}

func handleNotFound(args []string) (commandResult, commandStatus) {
	command := args[0]

	return fmt.Sprintf("%s: command not found", command), 1
}
