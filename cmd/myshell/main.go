package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type command func([]string)

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

		builtinCommand, ok := getBuiltin(commandName)

		if !ok {
			handleNotFound(inputs)
			continue
		}

		builtinCommand(inputs)
	}
}

func getBuiltin(commandName string) (command, bool) {
	var result command
	ok := false

	switch commandName {
	case "exit":
		result = handleExit
		ok = true
	case "type":
		result = handleType
		ok = true
	case "echo":
		result = handleEcho
		ok = true
	}

	return result, ok
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
	xs := args[1:]

	for _, x := range xs {
		_, ok := getBuiltin(x)

		if !ok {
			fmt.Printf("%s not found\n", x)
		} else {
			fmt.Printf("%s is a shell builtin\n", x)
		}
	}
}

func handleNotFound(args []string) {
	command := args[0]

	fmt.Printf("%s: command not found\n", command)
}
