package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	// Uncomment this block to pass the first stage
	fmt.Fprint(os.Stdout, "$ ")

	// Wait for user input
	reader, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err == io.EOF {
		fmt.Println("closing shell...")
	} else if err != nil {
		fmt.Println("error: ", err.Error())
	}

	inputs := strings.Split(strings.TrimSpace(reader), " ")

	if len(inputs) > 0 {
		command := inputs[0]

		switch command {
		default:
			output, _ := fmt.Printf("%s: command not found", command)
			fmt.Fprintln(os.Stdout, output)
		}
	}
}
