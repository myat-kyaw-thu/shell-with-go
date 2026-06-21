package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var builtins = map[string]bool{
	"echo": true,
	"exit": true,
	"type": true,
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

		if input == "exit" {
			os.Exit(0)
		} else if strings.HasPrefix(input, "echo ") {
			message := strings.TrimPrefix(input, "echo ")
			fmt.Println(message)
		} else if strings.HasPrefix(input, "type ") {
			arg := strings.TrimPrefix(input, "type ")
			if builtins[arg] {
				fmt.Printf("%s is a shell builtin\n", arg)
			} else {
				fmt.Printf("%s: not found\n", arg)
			}
		} else {
			fmt.Printf("%s: command not found\n", input)
		}
	}
}
