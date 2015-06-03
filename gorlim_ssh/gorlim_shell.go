package main

import (
	"bytes"
	"fmt"
	"github.com/gorilla/rpc/json"
	"github.com/gorlim/Gorlim/gorlim"
	"log/syslog"
	"net/http"
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

func onError(err error) bool {
	if err == nil {
		return false
	}
	logwriter, e := syslog.New(syslog.LOG_ERR|syslog.LOG_LOCAL0, "gorlim_shell")
	if e == nil {
		logwriter.Err(fmt.Sprintf("error(%v) with args(%v)", err.Error(), os.Args))
	}

	fmt.Fprintln(os.Stderr, "-------------------------------------------------------------------------------")
	fmt.Fprintln(os.Stderr, "| Gorlim service is temporary down.                                           |")
	fmt.Fprintln(os.Stderr, "| We've already sent 4 hobbits, 2 men, dwarf, elf and wizard to deal with it. |")
	fmt.Fprintln(os.Stderr, "| Thanks for your patience.                                                   |")
	fmt.Fprintln(os.Stderr, "-------------------------------------------------------------------------------")
	return true
}

func main() {
	user := os.Args[1]
	port := os.Args[2]
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
	if head == "git-upload-pack" {
		args := gorlim.CheckRepoArgs{
			RepoPath: parts[1],
		}

		response := gorlim.CheckRepoResponse{}
		for {
			b, err := json.EncodeClientRequest("GithubService.CheckRepo", args)
			if onError(err) {
				return
			}
			req, err := http.NewRequest("POST", "http://localhost:"+port, bytes.NewReader(b))
			if onError(err) {
				return
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if onError(err) {
				return
			}
			defer resp.Body.Close()
			if onError(json.DecodeClientResponse(resp.Body, &response)) {
				return
			}
			fmt.Fprintf(os.Stderr, "Last time this Gorlim repo was synced with web interface is %v\n", response.LastConvertedEventTime)
			if response.DoneRatio > 0 {
				break
			}
		}
	}
	parts = parts[1:len(parts)]

	child := exec.Command(head, parts...)
	child.Stdin = os.Stdin
	child.Stderr = os.Stderr
	child.Stdout = os.Stdout
	err := child.Run()
	onError(err)
}
