package main

import "os"
import "strings"
import "os/exec"

func main() {
	// read in input from stdin
	user := os.Args[0]
	// send to socket
	os.Setenv("GORLIM_USER", user)
	cmd := os.Getenv("SSH_ORIGINAL_COMMAND")
	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

	err := exec.Command(head, parts...).Run()
	if err != nil {
		panic(err)
	}
}
