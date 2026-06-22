package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type redirect struct {
	stdoutFile   string
	stdoutAppend bool
	stderrFile   string
	stderrAppend bool
}

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

func extractRedirect(parts []string) ([]string, redirect) {
	var r redirect
	var filtered []string
	for i := 0; i < len(parts); i++ {
		if (parts[i] == ">" || parts[i] == "1>") && i+1 < len(parts) {
			r.stdoutFile = parts[i+1]
			r.stdoutAppend = false
			i++
		} else if (parts[i] == ">>" || parts[i] == "1>>") && i+1 < len(parts) {
			r.stdoutFile = parts[i+1]
			r.stdoutAppend = true
			i++
		} else if parts[i] == "2>" && i+1 < len(parts) {
			r.stderrFile = parts[i+1]
			r.stderrAppend = false
			i++
		} else if parts[i] == "2>>" && i+1 < len(parts) {
			r.stderrFile = parts[i+1]
			r.stderrAppend = true
			i++
		} else {
			filtered = append(filtered, parts[i])
		}
	}
	return filtered, r
}

func openOutput(path string, append bool) (*os.File, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	return os.OpenFile(path, flags, 0644)
}

func runBuiltin(command string, args []string, r redirect) {
	out := os.Stdout
	if r.stdoutFile != "" {
		f, err := openOutput(r.stdoutFile, r.stdoutAppend)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer f.Close()
		out = f
	}
	errOut := os.Stderr
	if r.stderrFile != "" {
		f, err := openOutput(r.stderrFile, r.stderrAppend)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer f.Close()
		errOut = f
	}

	switch command {
	case "exit":
		os.Exit(0)
	case "echo":
		fmt.Fprintln(out, strings.Join(args, " "))
	case "pwd":
		if dir, err := os.Getwd(); err != nil {
			fmt.Fprintln(errOut, err)
		} else {
			fmt.Fprintln(out, dir)
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
			fmt.Fprintf(errOut, "cd: %s: No such file or directory\n", dir)
		}
	case "type":
		if len(args) == 0 {
			return
		}
		arg := args[0]
		switch {
		case builtins[arg]:
			fmt.Fprintf(out, "%s is a shell builtin\n", arg)
		case findInPath(arg) != "":
			fmt.Fprintf(out, "%s is %s\n", arg, findInPath(arg))
		default:
			fmt.Fprintf(out, "%s: not found\n", arg)
		}
	}
}

func runExternal(command string, args []string, r redirect) {
	path := findInPath(command)
	if path == "" {
		fmt.Printf("%s: command not found\n", command)
		return
	}
	cmd := exec.Command(path, args...)
	cmd.Args = append([]string{command}, args...)
	cmd.Stderr = os.Stderr

	if r.stdoutFile != "" {
		f, err := openOutput(r.stdoutFile, r.stdoutAppend)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer f.Close()
		cmd.Stdout = f
	} else {
		cmd.Stdout = os.Stdout
	}

	if r.stderrFile != "" {
		f, err := openOutput(r.stderrFile, r.stderrAppend)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer f.Close()
		cmd.Stderr = f
	} else {
		cmd.Stderr = os.Stderr
	}
	cmd.Run()
}

func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case escaped:
			current.WriteByte(ch)
			escaped = false
		case inSingle:
			if ch == '\'' {
				inSingle = false
			} else {
				current.WriteByte(ch)
			}
		case inDouble:
			if ch == '"' {
				inDouble = false
			} else if ch == '\\' && i+1 < len(input) && (input[i+1] == '"' || input[i+1] == '\\') {
				i++
				current.WriteByte(input[i])
			} else {
				current.WriteByte(ch)
			}
		case ch == '\\':
			escaped = true
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

		parts, r := extractRedirect(parts)
		if len(parts) == 0 {
			continue
		}

		command, args := parts[0], parts[1:]

		if builtins[command] {
			runBuiltin(command, args, r)
		} else {
			runExternal(command, args, r)
		}

	}
}
