package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var builtins = map[string]bool{
	"echo": true,
	"exit": true,
	"type": true,
	"pwd":  true,
	"cd":   true,
}

func findInPath(command string) string {
	for _, dir := range strings.Split(os.Getenv("PATH"), ":") {
		fullPath := dir + "/" + command
		if info, err := os.Stat(fullPath); err == nil && info.Mode()&0111 != 0 {
			return fullPath
		}
	}
	return ""
}

func runBuiltin(command string, args []string) {
	switch command {
	case "exit":
		os.Exit(0)

	case "echo":
		fmt.Println(strings.Join(args, " "))

	case "pwd":
		if dir, err := os.Getwd(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			fmt.Println(dir)
		}

	case "cd":
		if len(args) == 0 {
			return
		}
		dir := args[0]
		if dir == "~" {
			dir = os.Getenv("HOME")
		}
		if err := os.Chdir(dir); err != nil {
			fmt.Printf("cd: %s: No such file or directory\n", dir)
		}

	case "type":
		if len(args) == 0 {
			return
		}
		arg := args[0]
		switch {
		case builtins[arg]:
			fmt.Printf("%s is a shell builtin\n", arg)
		case findInPath(arg) != "":
			fmt.Printf("%s is %s\n", arg, findInPath(arg))
		default:
			fmt.Printf("%s: not found\n", arg)
		}
	}
}

func runExternal(command string, args []string) {
	path := findInPath(command)
	if path == "" {
		fmt.Printf("%s: command not found\n", command)
		return
	}
	cmd := exec.Command(path, args...)
	cmd.Args = append([]string{command}, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			} else {
				current.WriteByte(ch)
			}
		case inDouble:
			if ch == '"' {
				inDouble = false
			} else {
				current.WriteByte(ch)
			}
		case ch == '\'':
			inSingle = true
		case ch == '"':
			inDouble = true
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")

		input, err := reader.ReadString('\n')
		if err != nil {
			os.Exit(0)
		}

		parts := parseArgs(input)
		if len(parts) == 0 {
			continue
		}

		command, args := parts[0], parts[1:]

		if builtins[command] {
			runBuiltin(command, args)
		} else {
			runExternal(command, args)
		}
	}
}
