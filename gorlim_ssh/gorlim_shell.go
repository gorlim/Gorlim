package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func unescape(in []string) []string {
	for i := range in {
		s := in[i]
		first := s[0]
		last := s[len(s)-1]
		if first == '\'' && last == '\'' {
			in[i] = s[1 : len(s)-1]
		}
	}
	return in
}

func main() {
	// read in input from stdin
	user := os.Args[0]
	// send to socket
	os.Setenv("GORLIM_USER", user)
	cmd := os.Getenv("SSH_ORIGINAL_COMMAND")
	fmt.Fprintf(os.Stderr, "original command: %v\nargs: %v\n", cmd, os.Args)
	parts := unescape(strings.Fields(cmd))
	head := parts[0]
	parts = parts[1:len(parts)]

	child := exec.Command(head, parts...)
	child.Stdin = os.Stdin
	child.Stderr = os.Stderr
	child.Stdout = os.Stdout
	err := child.Run()
	if err != nil {
		panic(err)
	}
}
