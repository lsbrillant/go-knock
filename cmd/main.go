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

	var err error
	if len(args) > 0 {
		switch args[0] {
		case "send":
			err = Send("127.0.0.1", knocks...)
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		fmt.Printf("tell me to do something\n")
	}
}
