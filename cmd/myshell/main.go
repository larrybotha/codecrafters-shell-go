package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type (
	command     func([]string)
	commandType int
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
			fmt.Println("closing shell...")
		} else if err != nil {
			fmt.Println("error: ", err.Error())
		}

		inputs := strings.Split(strings.TrimSpace(reader), " ")
		commandName := ""

		if len(inputs) > 0 {
			commandName = strings.TrimSpace(inputs[0])
		}

		if commandName == "" {
			continue
		}

		builtinCommand, cmdType := getBuiltinCommand(commandName)

		if cmdType == cmdNotFound {
			handleNotFound(inputs)
			continue
		}

		builtinCommand(inputs)
	}
}

func getBuiltinCommand(commandName string) (command, commandType) {
	builtins := map[string]command{
		"exit": handleExit,
		"type": handleType,
		"echo": handleEcho,
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

func handleExit(args []string) {
	if len(args) > 2 {
		fmt.Print("too many arguments\n")

		return
	}

	status := 0

	if len(args) > 1 {
		arg := args[1]
		result, err := strconv.Atoi(arg)
		if err != nil {
			status = result
		}
	}

	os.Exit(status)
}

func handleEcho(args []string) {
	values := args[1:]

	fmt.Print(strings.Join(values, " ") + "\n")
}

func handleType(args []string) {
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
			fmt.Printf("%s is a shell builtin\n", cmd)
		case cmdSystem:
			fmt.Printf("%s is %s\n", filepath.Base(cmd), cmd)
		default:
			fmt.Printf("%s: not found\n", cmd)
		}
	}
}

func handleNotFound(args []string) {
	command := args[0]

	fmt.Printf("%s: command not found\n", command)
}
