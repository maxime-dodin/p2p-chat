package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

// User store arguments
type User struct {
	Host     string
	Port     int
	Username string
	args     []string
}

func handleStream(s network.Stream) {
	log.Println("Got a new stream!")

	// create a buff stream for non-blocking read and writes.
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
	go writeData(rw)
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m>", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}
		str := fmt.Sprintf("%s\n", sendData)
		rw.WriteString(str)
		rw.Flush()
	}
}

func dispatchUser(user *User, h host.Host) {
	if user.Host == "default" {
		startPeerAndHost(h, handleStream)
	} else {
		rw, err := startPeerAndConnect(h, user.Host)
		if err != nil {
			log.Println(err)
			return
		}
		// create a go-routine to read and write data.
		go writeData(rw)
		go readData(rw)
	}
}

func startPeerToPeer(user *User) {
	reader := new(io.Reader)
	//assign a random port
	*reader = rand.Reader

	h, err := makeHost(user.Port, *reader)
	if err != nil {
		log.Println(err)
		return
	}
	dispatchUser(user, h)
	select {} // block further execution
}

func argumentHandling(binName string, args []string) (u *User, output string, err error) {
	flags := flag.NewFlagSet(binName, flag.ContinueOnError)
	buf := new(bytes.Buffer)
	flags.SetOutput(buf)

	user := new(User)
	flags.StringVar(&user.Host, "host", "default", "host addr from the first client")
	flags.IntVar(&user.Port, "port", 8080, "port number to open")
	flags.StringVar(&user.Username, "username", "Anonymous", "your name during the chat")

	err = flags.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}
	user.args = flags.Args()
	return user, buf.String(), nil
}

func main() {
	user, output, err := argumentHandling(os.Args[0], os.Args[1:])

	if err == flag.ErrHelp {
		fmt.Println(output)
		os.Exit(2)
	} else if err != nil {
		fmt.Println("got error:", err)
		fmt.Println("output:\n", output)
		os.Exit(1)
	}
	startPeerToPeer(user)
}

func makeHost(port int, rand io.Reader) (host.Host, error) {
	// hash the secret key into a func
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// create adr listen by the nodes
	adr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))

	// create a new private p2p network
	return libp2p.New(
		libp2p.ListenAddrs(adr),
		libp2p.Identity(prvKey),
	)
}

func startPeerAndHost(h host.Host, streamHandler network.StreamHandler) {
	// set a new stream handler for incoming connections
	h.SetStreamHandler("/chat/1.0.0", streamHandler)

	// retrieve tcp port from adr
	var port string
	for _, la := range h.Network().ListenAddresses() {
		if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
			port = p
			break
		}
	}

	if port == "" {
		log.Println("was not able to find actual local port")
		return
	}

	log.Printf("Run './main -host /ip4/127.0.0.1/tcp/%v/p2p/%s' on another terminal.\n", port, h.ID().Pretty())
	log.Println("You can replace 127.0.0.1 with public IP for connection on different computer.")
	log.Println("Waiting for incoming connection...")
}

func startPeerAndConnect(h host.Host, hostAdr string) (*bufio.ReadWriter, error) {
	log.Println("This node's addresses:")
	for _, la := range h.Addrs() {
		log.Printf(" - %v\n", la)
	}
	log.Println()

	// turn the hostAdr into a readable adr.
	adr, err := multiaddr.NewMultiaddr(hostAdr)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// get p2p id from the adr.
	info, err := peer.AddrInfoFromP2pAddr(adr)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// store the adr/id, for further connections
	h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	// start a stream with the id and connect to the host.
	s, err := h.NewStream(context.Background(), info.ID, "/chat/1.0.0")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println("Established connection to destination")

	// create a buff stream for non-blocking read and writes.
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	return rw, nil
}
