package main

import "gorlim"
import "fmt"


func testPushSubscription() {
  pushevent := gorlim.GetPushListener()  
  gorlim.CreateRepo("." , 0)

  for push := range pushevent {
  	fmt.Println("push to repo id ", push)
  }
}

func main() {

//testPushSubscription()
  
}