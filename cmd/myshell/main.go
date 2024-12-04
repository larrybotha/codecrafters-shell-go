package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
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

		builtinCommand, cmdType := getBuiltinCaommdn(commandName)

		if cmdType == cmdNotFound {
			handleNotFound(inputs)
			continue
		}

		builtinCommand(inputs)
	}
}

func getBuiltinCaommdn(commandName string) (command, commandType) {
	var result command
	cmdType := cmdNotFound

	switch commandName {
	case "exit":
		result = handleExit
		cmdType = cmdBuiltin
	case "type":
		result = handleType
		cmdType = cmdBuiltin
	case "echo":
		result = handleEcho
		cmdType = cmdBuiltin
	}

	return result, cmdType
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
	results := make(map[string]commandType)

	for _, x := range commands {
		_, cmdType := getBuiltinCaommdn(x)

		results[x] = cmdType
	}

	for k, cmdType := range results {
		if cmdType == cmdNotFound {
			fmt.Printf("%s not found\n", k)
		} else {
			fmt.Printf("%s is a shell builtin\n", k)
		}
	}
}

func handleNotFound(args []string) {
	command := args[0]

	fmt.Printf("%s: command not found\n", command)
}
