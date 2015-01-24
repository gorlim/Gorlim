package gorlim

import "os"
import "time"
//import "fmt"

func SubscribeToPushEvent(pipe string, notify chan<- int) {
	//fmt.Println("Read byte")
	f, _ := os.Open(pipe)
  go func() {
  	for {
  	   bytes :=  make([]byte, 16)
  	   n, _ := f.Read(bytes)
         if n != 0 {
           //fmt.Println("Read byte", n)
           repoId := 0
           for i := 0; i < n - 1; i++ {
             repoId = repoId * 10 + int(bytes[i] - 48);
           }
           notify <- repoId // TODO
         } else { //ugly
         	//fmt.Println("Before sleep")
            time.Sleep(time.Second)
            //fmt.Println("After sleep")
        }
  	}
  }()
}