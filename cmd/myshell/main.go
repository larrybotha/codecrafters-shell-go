package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
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

		if len(inputs) > 0 {
			command := inputs[0]

			switch command {
			default:
				fmt.Printf("%s: command not found\n", command)
			}
		}
	}
}
