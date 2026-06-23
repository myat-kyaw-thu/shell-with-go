package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

type redirect struct {
	stdoutFile   string
	stdoutAppend bool
	stderrFile   string
	stderrAppend bool
}
type tabCompleter struct {
	builtins  []string
	lastInput string
	lastCount int
}

func (t *tabCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	input := string(line[:pos])

	if strings.Contains(input, " ") {
		prefix := input[strings.LastIndex(input, " ")+1:]

		dir := "."
		filePrefix := prefix
		if idx := strings.LastIndex(prefix, "/"); idx >= 0 {
			dir = prefix[:idx+1]
			filePrefix = prefix[idx+1:]
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, 0
		}
		var matches []string
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), filePrefix) {
				matches = append(matches, e.Name())
			}
		}
		sort.Strings(matches)

		if len(matches) == 0 {
			fmt.Fprint(os.Stdout, "\x07")
			return nil, 0
		}

		if len(matches) == 1 {
			completion := matches[0][len(filePrefix):]
			suffix := " "
			fullPath := dir + "/" + matches[0]
			if dir == "." {
				fullPath = matches[0]
			}
			if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
				suffix = "/"
			}
			return [][]rune{[]rune(completion + suffix)}, len(filePrefix)
		}

		// multiple matches
		if t.lastInput == input {
			t.lastCount++
		} else {
			t.lastInput = input
			t.lastCount = 1
		}

		if t.lastCount == 1 {
			fmt.Fprint(os.Stdout, "\x07")
			return nil, 0
		}

		// second tab: display matches with dir indicators
		var display []string
		for _, m := range matches {
			fullPath := dir + "/" + m
			if dir == "." {
				fullPath = m
			}
			if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
				display = append(display, m+"/")
			} else {
				display = append(display, m)
			}
		}
		fmt.Fprintf(os.Stdout, "\n%s\n$ %s", strings.Join(display, "  "), input)
		t.lastCount = 0
		return nil, 0
	}

	seen := map[string]bool{}
	all := append([]string{}, t.builtins...)
	for _, dir := range strings.Split(os.Getenv("PATH"), ":") {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				all = append(all, e.Name())
			}
		}
	}

	var matches []string
	for _, c := range all {
		if strings.HasPrefix(c, input) && !seen[c] {
			seen[c] = true
			matches = append(matches, c)
		}
	}
	sort.Strings(matches)

	if len(matches) == 0 {
		fmt.Fprint(os.Stderr, "\x07")
		return nil, 0
	}

	if len(matches) == 1 {
		completion := matches[0][len(input):]
		return [][]rune{[]rune(completion + " ")}, len(input)
	}

	lcp := matches[0]
	for _, m := range matches[1:] {
		for !strings.HasPrefix(m, lcp) {
			lcp = lcp[:len(lcp)-1]
		}
	}

	if len(lcp) > len(input) {
		completion := lcp[len(input):]
		t.lastInput = lcp
		t.lastCount = 0
		return [][]rune{[]rune(completion)}, len(input)
	}

	if t.lastInput == input {
		t.lastCount++
	} else {
		t.lastInput = input
		t.lastCount = 1
	}

	if t.lastCount == 1 {
		fmt.Fprint(os.Stderr, "\x07")
		return nil, 0
	}

	fmt.Fprintf(os.Stdout, "\n%s\n$ %s", strings.Join(matches, "  "), input)
	t.lastCount = 0
	return nil, 0
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
	completions := []string{"echo", "exit", "type", "pwd", "cd"}

	completer := &tabCompleter{builtins: completions}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "$ ",
		AutoComplete: completer,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer rl.Close()

	for {
		fmt.Print("$ ")

		input, err := rl.Readline()
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
