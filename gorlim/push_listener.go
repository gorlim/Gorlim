package gorlim

import (
	"bufio"
	"net"
	"strings"
	"fmt"
)

type EventType int

const ( 
	PrePushEvent EventType = 0
    PostPushEvent EventType = 1
)

type RepoEventReply struct {
	Status bool
	Message string
}

type RepoEventMessage struct {
	Event    EventType
	RepoPath string
	Sha      string
	Reply    chan RepoEventReply
}

type RepoEventListener struct {
	event chan RepoEventMessage
	exit  chan bool
}

func CreateRepoEventListener() RepoEventListener {
	listener := RepoEventListener{}
	listener.event = make(chan RepoEventMessage)
	listener.exit = make(chan bool)

	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		panic(err)
	}

	parseMessage := func (msg string) (op, repoPath, oldSha, newSha string) {
		fmt.Printf("message %s\n", msg)
		split := strings.Split(msg, " ")
		op = split[0]
		repoPath = split[1]
		oldSha = split[2]
		newSha = split[3]
		return
	}

	go func() {
		for {
			select {
			case _ = <-listener.exit:
				break
			default:
			}

			conn, err := ln.Accept()
			if err != nil {
				continue
			}

			go func() {
				defer conn.Close()

				message, _ := bufio.NewReader(conn).ReadString('\n')
				op, repoPath, _, newSha := parseMessage(message)
				ppm := RepoEventMessage{
					RepoPath: repoPath,
					Sha:      newSha[0:40],
					Reply:    make(chan RepoEventReply),
				}
				if op == "pre_push" {
					ppm.Event = PrePushEvent
				} else if op == "post_push" {
					ppm.Event = PostPushEvent
				} else {
				    conn.Write([]byte("Error: unknown hook kind\n"))
				    return
				}

				listener.event <- ppm
				reply := <-ppm.Reply
				if reply.Status == true {
					conn.Write([]byte("OK: " + reply.Message + "\n"))
				} else {
					conn.Write([]byte("Error: " + reply.Message + "\n"))
				}
			}()
		}
	}()

	return listener
}

func (ppl *RepoEventListener) GetRepoEventChannel() <-chan RepoEventMessage {
	return ppl.event
}

func (ppl *RepoEventListener) Close() {
	close(ppl.event)
	ppl.exit <- true
}
