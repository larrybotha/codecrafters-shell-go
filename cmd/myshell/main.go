package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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
		var command string
		if len(inputs) > 0 {
			command = strings.TrimSpace(inputs[0])
		}

		if command == "" {
			continue
		}

		switch command {
		case "exit":
			handleExit(inputs)
		default:
			fmt.Printf("%s: command not found\n", command)
		}
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
