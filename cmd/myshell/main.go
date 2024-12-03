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

var builtins = map[string]command{
	"exit":      handleExit,
	"echo":      handleEcho,
	"not_found": handleNotFound,
}

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
		var command string

		if len(inputs) > 0 {
			command = strings.TrimSpace(inputs[0])
		}

		if command == "" {
			continue
		}

		builtin, ok := builtins[command]
		if !ok {
			builtin = builtins["not_found"]
		}

		builtin(inputs)
	}
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

func handleNotFound(args []string) {
	command := args[0]

	fmt.Printf("%s: command not found\n", command)
}
