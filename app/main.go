package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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
		} else {
			fmt.Printf("%s: command not found\n", input)
		}
	}
}
