package main

import (
	"fmt"
	"log"
	"os"

	. "github.com/mentalpumkins/go-knock"
)

func main() {
	knocks := []Knock{
		Port(8081),
		Port(8082),
		Knock{
			Type:    "tcp",
			Port:    ":8080",
			PayLoad: []byte(":)"),
		},
		Port(8083),
		Port(8084),
	}

	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "listen":
			s, err := Listen(knocks...)
			if err != nil {
				log.Fatal(err)
			}
			select {
			case ip := <-s:
				fmt.Printf("WOOOOOO!! %s has Knocked\n", ip)
			}
		case "send":
			Send("localhost", knocks...)
		}
	} else {
		fmt.Printf("dont fucked up")
	}
}
