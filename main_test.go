package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	_test "github.com/libp2p/go-libp2p/examples/testutils"
	"log"
	"strings"
	"testing"
)

func TestArgumentHandlingValid(t *testing.T) {
	var tests = []struct {
		args []string
		user User
	}{
		{[]string{},
			User{Host: "default", Port: 8080, Username: "Anonymous", args: []string{}}},
		{[]string{"-port", "8000", "-username", "Maxime D"},
			User{Host: "default", Port: 8000, Username: "Maxime D", args: []string{}}},
		{[]string{"-port", "8000", "-username", "Maxime D"},
			User{Host: "default", Port: 8000, Username: "Maxime D", args: []string{}}},
		{[]string{"-host", "/ip4/127.0.0.1/tcp/8080/p2p/QmSyD8a7ww7mPvJj34oDDY49kwzoe9QAp4zjBUGJcEZuv5", "-username", "Thomas"},
			User{Host: "/ip4/127.0.0.1/tcp/8080/p2p/QmSyD8a7ww7mPvJj34oDDY49kwzoe9QAp4zjBUGJcEZuv5", Port: 8000, Username: "Thomas", args: []string{}}},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			_, output, err := argumentHandling("main", test.args)
			if err != nil {
				t.Errorf("err got %v, want nil", err)
			}
			if output != "" {
				t.Errorf("output got %q, want empty", output)
			}
		})
	}
}

func TestPeerToPeer(t *testing.T) {
	var h _test.LogHarness
	h.Expect("Waiting for incoming connection...")
	h.Expect("Established connection to destination")
	h.Expect("Got a new stream!")

	h.Run(t, func() {
		// create a temporary context to stop the host when
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// find a free tcp port
		port1, err := _test.FindFreePort(t, "", 3)
		if err != nil {
			log.Println(err)
			return
		}

		// find a free tcp port
		port2, err := _test.FindFreePort(t, "", 3)
		if err != nil {
			log.Println(err)
			return
		}

		h1, err := makeHost(port1, rand.Reader)
		if err != nil {
			log.Println(err)
			return
		}

		go startPeerAndHost(h1, func(network.Stream) {
			log.Println("Got a new stream!")
			cancel()
		})

		dest := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/p2p/%s", port1, h1.ID().Pretty())

		h2, err := makeHost(port2, rand.Reader)
		if err != nil {
			log.Println(err)
			return
		}

		go func() {
			rw, err := startPeerAndConnect(h2, dest)
			if err != nil {
				log.Println(err)
				return
			}

			rw.WriteString("Hi, i am a test message")
			rw.Flush()
		}()

		<-ctx.Done()
	})
}
