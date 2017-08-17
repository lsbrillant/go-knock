// Go implementation of port knocking.
package knock

import (
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Knocker interface {
	Knock(...Knock)
}

// A single knock to either listen for or execute.
type Knock struct {
	// either udp or tcp
	Type string
	// listen on
	Port string
	// bytes to send
	PayLoad []byte
}

type Listener struct {
	Host string
}

func Port(number int) Knock {
	return Knock{
		Type:    "tcp",
		Port:    ":" + strconv.Itoa(number),
		PayLoad: []byte{},
	}
}

func Listen(knocks ...Knock) (successes chan string, err error) {
	attempts := make(map[string]int)
	lock := sync.RWMutex{}
	var end int = len(knocks)
	successes = make(chan string, 5)
	for i, knock := range knocks {
		var ln net.Listener
		step := i + 1
		ln, err = net.Listen(knock.Type, knock.Port)
		if err != nil {
			return
		}
		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					continue
				}
				go func() {
					defer conn.Close()
					lock.RLock()
					source := strings.Split(conn.RemoteAddr().String(), ":")[0]
					stage := attempts[source]
					lock.RUnlock()
					lock.Lock()
					switch {
					case stage == step-1:
						recv, err := ioutil.ReadAll(conn)
						if err != nil {
							// TODO: something ?
						}
						if byteEqual(recv, knocks[step-1].PayLoad) {
							attempts[source], stage = step, step
							break
						}
						fallthrough
					default:
						delete(attempts, source)
					}
					lock.Unlock()
					if stage == end {
						successes <- source
					}
					log.Print(source)
				}()
			}
		}()
	}
	return
}

func Send(host string, knocks ...Knock) {
	for _, knock := range knocks {
	retry:
		conn, err := net.Dial(knock.Type, host+knock.Port)
		if err != nil {
			conn.Close()
			goto retry
		}
		conn.Write(knock.PayLoad)
		conn.Close()
	}
}
