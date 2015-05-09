package main

import (
	"os"
	"os/exec"
	"strings"
)

var approved = map[string]string{
	"git-upload-pack": "git-upload-pack",
}

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
	user := os.Args[0]
	os.Setenv("GORLIM_USER", user)
	cmd := os.Getenv("SSH_ORIGINAL_COMMAND")

	parts := unescape(strings.Fields(cmd))
	if len(parts) == 0 {
		return
	}
	head := parts[0]
	if _, ok := approved[head]; !ok {
		return
	}
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
