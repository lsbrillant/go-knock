package main

import (
	"fmt"
	"log"
	"os"

	. "github.com/lsbrillant/go-knock"
)

func main() {
	knocks := []Knock{
		Port(8081),
		Port(8082),
		PayLoad(8080, []byte(":)")),
		Port(8083),
		Port(8084),
	}

	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "listen":
			ln, err := ListenCtx(knocks...)
			if err != nil {
				log.Fatal(err)
			}
			ip := ln.Accept()
			fmt.Printf("%s is the one who knocks\n", ip)
			ln.Close()
			for {

			}
		case "send":
			Send("localhost", knocks...)
		}
	} else {
		fmt.Printf("tell me to do something\n")
	}
}
