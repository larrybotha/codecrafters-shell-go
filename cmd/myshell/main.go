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
	cmdCompound
	cmdNotFound
	cmdSystem
)

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")

		input, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err == io.EOF {
			fmt.Fprintln(os.Stdout, "closing shell...")
			os.Exit(0)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "error: ", err.Error())
		}

		handleInput(input)
	}
}

func handleInput(input string) {
	var lastOutput string
	args := parseArgs(strings.TrimSpace(input))
	aggregatedArgs := aggregateArgs(args)

	for i, aggregate := range aggregatedArgs {
		if len(lastOutput) > 0 {
			aggregate = append(aggregate, lastOutput)
		}

		if len(aggregate) == 0 {
			continue
		}

		commandName := strings.TrimSpace(aggregate[0])
		output, status := executeCommand(commandName, aggregate)
		output = strings.TrimSpace(output)

		switch true {
		case status > 0:
			fmt.Fprintln(os.Stderr, output)
		case i == len(aggregatedArgs)-1 && len(output) > 0:
			fmt.Fprintln(os.Stdout, output)
		default:
			lastOutput = output
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
	var currWord strings.Builder
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
				startNextWord = startNextWord || currWord.Len() > 0
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

		currWord.WriteString(toAdd)

		if startNextWord {
			args = append(args, strings.TrimSpace(currWord.String()))

			currWord.Reset()
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
			currAggregate = []string{x}
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
	if compoundCommand, cmdType := getCompoundCommand(commandName); cmdType == cmdCompound {
		return compoundCommand(inputs)
	}

	if builtinCommand, cmdType := getBuiltinCommand(commandName); cmdType == cmdBuiltin {
		return builtinCommand(inputs)
	}

	if cmdPath, cmdType := getSystemCommand(commandName); cmdType == cmdSystem {
		return execSystemCommand(cmdPath, inputs)
	}

	return handleNotFound(inputs)
}

func execSystemCommand(cmdPath string, inputs []string) (commandOutput, commandStatus) {
	var status commandStatus = 1
	var output commandOutput
	args := inputs[1:]
	cmd := exec.Command(cmdPath, args...)

	cmdOutput, err := cmd.CombinedOutput()
	if err == nil {
		status = 0
	}

	output = string(cmdOutput)

	return output, status
}

func handleRedirect(args []string) (commandOutput, commandStatus) {
	var output commandOutput
	var file *os.File
	var err error
	status := 1

	if len(args) < 2 {
		output = fmt.Sprintf("too few arguments for redirect")

		return output, status
	}

	fileName := args[1]
	fileArgs := args[2:]

	_, statErr := os.Stat(fileName)

	if os.IsNotExist(statErr) {
		file, err = os.Create(fileName)
	} else {
		file, err = os.OpenFile(fileName, os.O_WRONLY, 0o644)
	}

	if file != nil && err == nil {
		_, err = file.Write([]byte(strings.Join(fileArgs, "")))
	}

	if err != nil {
		output = err.Error()
	} else {
		status = 0
	}

	return output, status
}

// TODO: should return err if commandName is not compound
func getCompoundCommand(commandName string) (command, commandType) {
	var cmd command
	cmdType := cmdNotFound

	if strings.HasSuffix(commandName, ">") {
		cmdType = cmdCompound
		cmd = handleRedirect
	}

	return cmd, cmdType
}

// TODO: should return err if commandName is not compound
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

// TODO: should return err if commandName is not compound
func getSystemCommand(commandName string) (string, commandType) {
	var path string
	paths := strings.Split(os.Getenv("PATH"), ":")
	cmdType := cmdNotFound

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

	// TODO: understand why this doesn't raise compiler errors when followed by
	// return statement
	os.Exit(status)

	return output, status
}

func handleCd(args []string) (commandOutput, commandStatus) {
	var status commandStatus = 1
	path := strings.TrimSpace(strings.Join(args[1:], ""))
	output := fmt.Sprintf("cd: %s: No such file or directory", path)

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			output = err.Error()
		}

		path = homeDir
	}

	if err := os.Chdir(path); err == nil {
		status = 0
		output = ""
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
