package gorlim

import (
	"bufio"
	"net"
	"strings"
)

type RepoPrePushMessage struct {
	RepoPath string
	Sha      string
}

type RepoPrePushReply struct {
	Status bool
	Err    string
}

type PrePushListener struct {
	event chan RepoPrePushMessage
	reply chan RepoPrePushReply
	exit  chan bool
}

func CreatePrePushListener() PrePushListener {
	listener := PrePushListener{}
	listener.event = make(chan RepoPrePushMessage)
	listener.exit = make(chan bool)
	listener.reply = make(chan RepoPrePushReply)

	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		panic(err)
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
				message, _ := bufio.NewReader(conn).ReadString('\n')
				split := strings.Split(message, " ")
				ppm := RepoPrePushMessage{
					RepoPath: split[0],
					Sha:      split[2][0:40],
				}
				listener.event <- ppm
				reply := <-listener.reply
				if reply.Status == true {
					conn.Write([]byte("OK\n"))
				} else {
					conn.Write([]byte("Error: " + reply.Err + "\n"))
				}
			}()
		}
	}()

	return listener
}

func (ppl *PrePushListener) GetPrePushChannel() <-chan RepoPrePushMessage {
	return ppl.event
}

func (ppl *PrePushListener) GetReplyChannel() chan<- RepoPrePushReply {
	return ppl.reply
}

func (ppl *PrePushListener) Close() {
	close(ppl.event)
	ppl.exit <- true
}
