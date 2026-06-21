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
}

func findInPath(command string) string {
	pathEnv := os.Getenv("PATH")
	dirs := strings.Split(pathEnv, ":")

	for _, dir := range dirs {
		fullPath := dir + "/" + command

		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if info.Mode()&0111 != 0 {
			return fullPath
		}
	}
	return ""
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")

		input, err := reader.ReadString('\n')
		if err != nil {
			os.Exit(0)
		}

		input = strings.TrimSpace(input)

		parts := strings.Fields(input)

		if len(parts) == 0 {
			continue
		}

		command := parts[0]
		args := parts[1:]

		if input == "exit" {
			os.Exit(0)
		} else if strings.HasPrefix(input, "echo ") {
			message := strings.TrimPrefix(input, "echo ")
			fmt.Println(message)
		} else if strings.HasPrefix(input, "type ") {
			arg := strings.TrimPrefix(input, "type ")
			if builtins[arg] {
				fmt.Printf("%s is a shell builtin\n", arg)
			} else if path := findInPath(arg); path != "" {
				fmt.Printf("%s is %s\n", arg, path)
			} else {
				fmt.Printf("%s: not found\n", arg)
			}
		} else if path := findInPath(command); path != "" {
			cmd := exec.Command(path, args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		} else {
			fmt.Printf("%s: command not found\n", input)
		}
	}
}
