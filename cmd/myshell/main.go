package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
			fmt.Fprintln(os.Stdout, "closing shell...")
			os.Exit(0)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "error: ", err.Error())
		}

		inputs := getArgs(strings.TrimSpace(reader))

		if len(inputs) == 0 {
			continue
		}

		commandName := strings.TrimSpace(inputs[0])

		executeCommand(commandName, inputs)
	}
}

func getArgs(input string) []string {
	var args []string

	r := regexp.MustCompile(`'[^'][\\.][\\.]*'|"[^"][\\.]*"|\S+`)

	if result := r.FindAllString(input, -1); result != nil {
		for _, arg := range result {
			args = append(args, prepareArg(arg))
		}
	}

	return args
}

const (
	quotesOpen = iota
	quotesClosed
)

func prepareArg(rawArg string) string {
	var firstChar byte

	if len(rawArg) > 0 {
		firstChar = rawArg[0]
	}

	switch firstChar {
	case '"':
		var tmpString []rune
		maybeEscaping := false

		for _, x := range rawArg {
			var ys []rune

			switch x {
			case '"':
				if maybeEscaping {
					ys = append(ys, x)
					maybeEscaping = false
				}
			case '\\':
				if maybeEscaping {
					ys = append(ys, x)
				}

				maybeEscaping = !maybeEscaping
			case '$', 'n', '`':
				if maybeEscaping {
					maybeEscaping = false
				}

				tmpString = append(tmpString, x)
			default:
				if maybeEscaping {
					ys = append(ys, '\\')
					maybeEscaping = false
				}

				ys = append(ys, x)
			}

			tmpString = append(tmpString, ys...)
		}

		rawArg = string(tmpString)
	case '\'':
		var tmpString string

		for _, x := range strings.Split(rawArg, "") {
			if x == "'" {
				continue
			}

			tmpString += x
		}

		rawArg = tmpString
	default:
		rawArg = strings.ReplaceAll(rawArg, "\\", "")
	}
	return rawArg
}

func executeCommand(commandName string, inputs []string) {
	if builtinCommand, cmdType := getBuiltinCommand(commandName); cmdType == cmdBuiltin {
		builtinCommand(inputs)
		return
	}

	if cmdPath, cmdType := getSystemCommand(commandName); cmdType == cmdSystem {
		args := inputs[1:]
		cmd := exec.Command(cmdPath, args...)

		if output, err := cmd.Output(); err != nil {
			log.Fatal(err.Error())
		} else {
			fmt.Fprint(os.Stdout, string(output))
		}

		return
	}

	handleNotFound(inputs)
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
		fmt.Fprint(os.Stderr, "too many arguments\n")

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

func handleCd(args []string) {
	path := strings.TrimSpace(strings.Join(args[1:], ""))

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			return
		}

		path = homeDir
	} else if fileInfo, err := os.Stat(path); err != nil || !fileInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", path)
		return
	}

	os.Chdir(path)
}

func handleEcho(args []string) {
	values := args[1:]

	fmt.Fprint(os.Stdout, strings.Join(values, " ")+"\n")
}

func handlePwd(args []string) {
	if len(args[1:]) > 2 {
		fmt.Fprint(os.Stderr, "too many arguments\n")

		return
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error()+"\n")
		return
	}

	fmt.Fprint(os.Stdout, wd+"\n")
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
			fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", cmd)
		case cmdSystem:
			fmt.Fprintf(os.Stdout, "%s is %s\n", filepath.Base(cmd), cmd)
		default:
			fmt.Fprintf(os.Stderr, "%s: not found\n", cmd)
		}
	}
}

func handleNotFound(args []string) {
	command := args[0]

	fmt.Fprintf(os.Stderr, "%s: command not found\n", command)
}
