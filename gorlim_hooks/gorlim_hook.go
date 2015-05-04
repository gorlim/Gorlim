package main

import "net"
import "fmt"
import "bufio"
import "os"
import "strings"

func main() {
  // connect to this socket
  conn, err := net.Dial("tcp", "127.0.0.1:8081")
  if err != nil {
  	fmt.Print("Error in hook handler: failed to connect to Gorlim")
  	os.Exit(1)
  }
  defer conn.Close()
  // read in input from stdin
  op := os.Args[1]
  id := os.Args[2]
  shaOld := os.Args[3]
  shaNew := os.Args[4]
  // send to socket
  message := op + " " + id + " " + shaOld + " " + shaNew + "\n";
  fmt.Fprintf(conn, message)
  // listen for reply
  message, err = bufio.NewReader(conn).ReadString('\n')
  if err != nil {
  	fmt.Print("Error in hook handler: failed to get reply from Gorlim\n")
  	os.Exit(1)
  }
  split := strings.SplitN(message, " ", 2)
  status, text := split[0], split[1]
  fmt.Println(text)
  if (status[0:2] != "OK") {
  	os.Exit(1)
  }
}