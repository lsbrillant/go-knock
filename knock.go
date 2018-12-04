// Go implementation of port knocking.
package knock

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

// A single knock to either listen for or execute.
type Knock struct {
	// either udp or tcp
	Type string
	// port to knock/listen on
	Port int
	// bytes to send
	PayLoad []byte
}

func Port(number int) Knock {
	return Knock{
		Type:    "tcp",
		Port:    number,
		PayLoad: []byte{},
	}
}

func PayLoad(port int, payload []byte) Knock {
	return Knock{
		Type:    "tcp",
		Port:    port,
		PayLoad: payload,
	}
}

// returns a basic Knock on port numbered number

// listen for knocks, the channel returned gives ips that have successfully
// knocked the knocks.
// TODO
// 	- better sync
//  - context based cancel
func Listen(knocks ...Knock) (successes chan string, err error) {
	attempts := make(map[string]int)
	lock := sync.RWMutex{}
	var end int = len(knocks)
	successes = make(chan string, 5)
	for i, knock := range knocks {
		var ln net.Listener
		step := i + 1
		ln, err = net.Listen(knock.Type, ":"+strconv.Itoa(knock.Port))
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
						recv := make([]byte, len(knocks[step-1].PayLoad))
						_, err := conn.Read(recv)
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
					if stage == end {
						successes <- source
					}
					lock.Unlock()
					//log.Printf("%s at %s", source, knocks[step-1].Port)
				}()
			}
		}()
	}
	return
}

type Listener interface {
	Accept() string
	Close()
}

type knockListener struct {
	successes  chan string
	cancelFunc func()
}

func (l *knockListener) Accept() string { return <-l.successes }
func (l *knockListener) Close()         { l.cancelFunc() }

func ListenCtx(knocks ...Knock) (Listener, error) {
	ctx, cancel := context.WithCancel(context.Background())
	listener := new(knockListener)

	successes := make(chan string, 5)

	listener.successes = successes
	listener.cancelFunc = cancel

	attempts := make(map[string]int)
	lock := sync.RWMutex{}
	var end int = len(knocks)
	var ln net.Listener
	var err error
	for i, knock := range knocks {
		step := i + 1
		ln, err = net.Listen(knock.Type, ":"+strconv.Itoa(knock.Port))
		if err != nil {
			return nil, err
		}
		go func(ctx context.Context, knock Knock) {
			log.Printf("listening on port %d", knock.Port)
			for {
				conn, err := ln.Accept()
				if err != nil {
					continue
				}
				select {
				case <-ctx.Done():
					log.Printf("exit listen on port %d", knock.Port)
					ln.Close()
					return
				default:
					go func() {
						defer conn.Close()
						lock.RLock()
						source := strings.Split(conn.RemoteAddr().String(), ":")[0]
						stage := attempts[source]
						lock.RUnlock()
						lock.Lock()
						switch {
						case stage == step-1:
							recv := make([]byte, len(knock.PayLoad))
							_, err := conn.Read(recv)
							if err != nil {
								// TODO: something ?
							}
							if byteEqual(recv, knock.PayLoad) {
								attempts[source], stage = step, step
								break
							}
							fallthrough
						default:
							delete(attempts, source)
						}
						if stage == end {
							listener.successes <- source
						}
						lock.Unlock()
						log.Printf("%s at %d", source, knock.Port)
					}()
				}
			}
		}(ctx, knock)
	}

	return listener, nil
}

// Sends knocks to host
func Send(host string, knocks ...Knock) {
	for _, knock := range knocks {
	retry:
		conn, err := net.Dial(knock.Type, host+":"+strconv.Itoa(knock.Port))
		if err != nil {
			conn.Close()
			goto retry
		}
		conn.Write(knock.PayLoad)
		conn.Close()
	}
}
