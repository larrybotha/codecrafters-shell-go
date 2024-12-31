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
	command         func(executionConfig) executionConfig
	commandType     int
	compoundCommand func(config executionConfig, prevConfig executionConfig) executionConfig
)

type (
	commandOutput = string
	commandStatus = int
)

type executionConfig struct {
	args        []string
	commandName string
	status      commandStatus
	stdErr      string
	stdOut      string
}

func newExecutionConfig(commandName string, args []string) executionConfig {
	return executionConfig{
		status:      1,
		args:        args,
		commandName: commandName,
	}
}

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
	prevExecution := newExecutionConfig("", []string{})
	args := parseArgs(strings.TrimSpace(input))
	aggregatedArgs := aggregateArgs(args)

	for _, aggregate := range aggregatedArgs {
		if len(aggregate) == 0 {
			continue
		}

		commandName := strings.TrimSpace(aggregate[0])
		execution := newExecutionConfig(commandName, aggregate)
		prevExecution = executeCommand(execution, prevExecution)
	}

	fd := os.Stderr
	output := strings.TrimSpace(prevExecution.stdErr)
	status := prevExecution.status

	if status == 0 {
		fd = os.Stdout
		output = strings.TrimSpace(prevExecution.stdOut)
	}

	if len(output) > 0 {
		fmt.Fprintln(fd, output)
	}
}

func isRedirect(x string) bool {
	return strings.HasSuffix(x, ">")
}

func aggregateArgs(args []string) [][]string {
	var aggregates [][]string
	currAggregate := []string{}

	for i, x := range args {
		if isRedirect(x) {
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

func executeCommand(config executionConfig, prevConfig executionConfig) executionConfig {
	if compoundCommand, cmdType := getCompoundCommand(config.commandName); cmdType == cmdCompound {
		return compoundCommand(config, prevConfig)
	}

	if builtinCommand, cmdType := getBuiltinCommand(config.commandName); cmdType == cmdBuiltin {
		return builtinCommand(config)
	}

	if cmdPath, cmdType := getSystemCommand(config.commandName); cmdType == cmdSystem {
		config.commandName = cmdPath
		return execSystemCommand(config)
	}

	return handleNotFound(config)
}

func execSystemCommand(config executionConfig) executionConfig {
	var stdErr strings.Builder
	cmdPath := config.commandName
	cmd := exec.Command(cmdPath, config.args[1:]...)
	cmd.Stderr = &stdErr

	cmdOutput, err := cmd.Output()
	if err == nil {
		config.status = 0
	} else {
		config.stdErr = stdErr.String()
	}

	config.stdOut = string(cmdOutput)

	return config
}

func handleRedirect(config executionConfig, prevConfig executionConfig) executionConfig {
	var file *os.File
	var err error

	if len(config.args) < 2 {
		config.stdOut = fmt.Sprintf("too few arguments for redirect")

		return config
	}

	fileName := config.args[1]
	fileArgs := []string{prevConfig.stdOut}
	fileArgs = append(fileArgs, config.args[2:]...)

	if len(prevConfig.stdErr) > 0 {
		fmt.Fprint(os.Stderr, prevConfig.stdErr)
	}

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
		config.stdErr = err.Error()
	} else {
		config.status = 0
	}

	return config
}

// TODO: should return err if commandName is not compound
func getCompoundCommand(commandName string) (compoundCommand, commandType) {
	var cmd compoundCommand
	cmdType := cmdNotFound

	if isRedirect(commandName) {
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

func handleExit(config executionConfig) executionConfig {
	args := config.args

	if len(args) > 2 {
		config.stdErr = fmt.Sprintf("too many arguments")

		return config
	}

	if len(args) > 1 {
		arg := args[1]

		if x, err := strconv.Atoi(arg); err != nil {
			config.stdErr = err.Error()
		} else {
			config.status = x
		}
	}

	// TODO: understand why this doesn't raise compiler errors when followed by
	// return statement
	os.Exit(config.status)

	return config
}

func handleCd(config executionConfig) executionConfig {
	args := config.args
	path := strings.TrimSpace(strings.Join(args[1:], ""))

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			config.stdErr = err.Error()
		}

		path = homeDir
	}

	if err := os.Chdir(path); err == nil {
		config.status = 0
	} else {
		config.stdErr = fmt.Sprintf("cd: %s: No such file or directory", path)
	}

	return config
}

func handleEcho(config executionConfig) executionConfig {
	config.status = 0
	config.stdOut = strings.Join(config.args[1:], " ")

	return config
}

func handlePwd(config executionConfig) executionConfig {
	args := config.args

	if len(args[1:]) > 2 {
		config.stdErr = "too many arguments"

		return config
	}

	wd, err := os.Getwd()
	if err != nil {
		config.stdErr = err.Error()
	} else {
		config.stdOut = wd
		config.status = 0
	}

	return config
}

func handleType(config executionConfig) executionConfig {
	args := config.args
	commands := args[1:]
	typeByCommand := make(map[string]commandType)

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
			config.stdOut = fmt.Sprintf("%s is a shell builtin", cmd)
			config.status = 0
		case cmdSystem:
			config.stdOut = fmt.Sprintf("%s is %s", filepath.Base(cmd), cmd)
			config.status = 0
		default:
			config.stdErr = fmt.Sprintf("%s: not found", cmd)
		}
	}

	return config
}

func handleNotFound(config executionConfig) executionConfig {
	command := config.args[0]
	config.stdErr = fmt.Sprintf("%s: command not found", command)

	return config
}
