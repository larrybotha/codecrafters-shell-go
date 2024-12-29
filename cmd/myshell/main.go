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
	command     func([]string) (commandOutput, commandStatus)
	commandType int
)

type (
	commandOutput = string
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
		aggregatedArgs := aggregateArgs(args)
		var lastOutput string

		for i, aggregate := range aggregatedArgs {
			aggregate = append(aggregate, lastOutput)

			if len(aggregate) == 0 {
				continue
			}

			commandName := strings.TrimSpace(aggregate[0])

			output, status := executeCommand(commandName, aggregate)

			if status > 0 {
				fmt.Fprintln(os.Stderr, output)
				break
			}

			if i == len(aggregatedArgs)-1 {
				fmt.Fprintln(os.Stdout, output)
			} else {
				lastOutput = output
			}
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

func aggregateArgs(args []string) [][]string {
	var aggregates [][]string
	currAggregate := []string{}

	for i, x := range args {
		if strings.HasSuffix(x, ">") {
			aggregates = append(aggregates, currAggregate)
			currAggregate = []string{}
		} else {
			currAggregate = append(currAggregate, x)
		}

		if i == len(args)-1 {
			aggregates = append(aggregates, currAggregate)
		}
	}

	return aggregates
}

func executeCommand(commandName string, inputs []string) (commandOutput, commandStatus) {
	if builtinCommand, cmdType := getBuiltinCommand(commandName); cmdType == cmdBuiltin {
		return builtinCommand(inputs)
	}

	if cmdPath, cmdType := getSystemCommand(commandName); cmdType == cmdSystem {
		var status commandStatus = 1
		var output commandOutput
		args := inputs[1:]
		cmd := exec.Command(cmdPath, args...)

		if xs, err := cmd.Output(); err != nil {
			output = err.Error()
		} else {
			output = string(xs)
		}

		return output, status
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
	output, ok := builtins[commandName]
	if ok {
		cmdType = cmdBuiltin
	}

	return output, cmdType
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

func handleExit(args []string) (commandOutput, commandStatus) {
	var output commandOutput
	status := 0

	if len(args) > 2 {
		output = fmt.Sprintf("too many arguments")
		status = 1
	}

	if len(args) > 1 {
		arg := args[1]
		x, err := strconv.Atoi(arg)
		if err != nil {
			status = x
			output = err.Error()
		}
	}

	// TODO: understand why this doesn't raise compiler errors
	os.Exit(status)

	return output, status
}

func handleCd(args []string) (commandOutput, commandStatus) {
	var status commandStatus = 1
	var output commandOutput
	path := strings.TrimSpace(strings.Join(args[1:], ""))

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			output = err.Error()
		}

		path = homeDir
	} else if fileInfo, err := os.Stat(path); err != nil || !fileInfo.IsDir() {
		output = fmt.Sprintf("cd: %s: No such file or directory", path)
	}

	err := os.Chdir(path)
	if err != nil {
		output = fmt.Sprintf("error: %s", err.Error())
	}

	if len(output) == 0 {
		status = 0
	}

	return output, status
}

func handleEcho(args []string) (commandOutput, commandStatus) {
	status := 0
	values := args[1:]
	output := strings.Join(values, " ")

	return output, commandStatus(status)
}

func handlePwd(args []string) (commandOutput, commandStatus) {
	var status commandStatus = 1
	var output commandOutput

	if len(args[1:]) > 2 {
		output = "too many arguments"

		return output, status
	}

	wd, err := os.Getwd()
	if err != nil {
		output = err.Error()
	} else {
		output = wd
		status = 0
	}

	return output, status
}

func handleType(args []string) (commandOutput, commandStatus) {
	commands := args[1:]
	typeByCommand := make(map[string]commandType)
	var status commandStatus = 1
	var output commandOutput

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
			output = fmt.Sprintf("%s is a shell builtin", cmd)
			status = 0
		case cmdSystem:
			output = fmt.Sprintf("%s is %s", filepath.Base(cmd), cmd)
			status = 0
		default:
			output = fmt.Sprintf("%s: not found", cmd)
		}
	}

	return output, status
}

func handleNotFound(args []string) (commandOutput, commandStatus) {
	command := args[0]

	return fmt.Sprintf("%s: command not found", command), 1
}
