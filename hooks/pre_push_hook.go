package main

import "net"
import "fmt"
import "bufio"
import "os"

func main() {
  // connect to this socket
  conn, err := net.Dial("tcp", "127.0.0.1:8081")
  if err != nil {
  	fmt.Print("Error in pre-push-hook: failed to connect to Gorlim")
  	os.Exit(1)
  }
  defer conn.Close()
  // read in input from stdin
  id := os.Args[1]
  shaOld := os.Args[2]
  shaNew := os.Args[3]
  // send to socket
  message := id + " " + shaOld + " " + shaNew + "\n";
  fmt.Fprintf(conn, message)
  // listen for reply
  message, err = bufio.NewReader(conn).ReadString('\n')
  if err != nil {
  	fmt.Print("Error in pre-push-hook: failed to get reply from Gorlim\n")
  	os.Exit(1)
  }
  if (message[0:2] != "OK") {
  	os.Exit(1)
  }
}