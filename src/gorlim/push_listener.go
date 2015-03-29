package gorlim

import "net"
import "bufio"
import "fmt"
import "strconv"
import "strings"

type RepoPrePushMessage struct {
  RepoId int
  Sha string
}

type RepoPrePushReply struct {
  Status bool
  Err string
}

type PrePushListener struct {
  event chan RepoPrePushMessage
  reply chan RepoPrePushReply
  exit chan bool
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
      if (err != nil) {
        continue
      }

      go func() {
        message, _ := bufio.NewReader(conn).ReadString('\n')
        fmt.Print("Message Received:", string(message))
        split := strings.Split(message, " ")
        id, _ := strconv.Atoi(split[0])
        ppm := RepoPrePushMessage{
          RepoId: id,
          Sha: split[2],
        }
        listener.event <- ppm
        reply := <- listener.reply
        if reply.Status {
          conn.Write([]byte("OK\n"))  
        } else {
          conn.Write([]byte("Error: " + reply.Err + "\n"))  
        }       
      } ();
    }
  } ();

  return listener;
}

func (ppl* PrePushListener) GetPrePushChannel() <-chan RepoPrePushMessage {
  return ppl.event;
}

func (ppl* PrePushListener) GetReplyChannel() chan<- RepoPrePushReply {
  return ppl.reply;
}

func (ppl* PrePushListener) Close() {
  close(ppl.event)
  ppl.exit <- true
}

