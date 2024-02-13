package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const ProtocolID = "/chat/1.0.0"

var streams []network.Stream

func handleStream(s network.Stream) {
    streams = append(streams, s)
}

func writeMessage(s network.Stream) {
    reader := bufio.NewReader(os.Stdin)

    for {
        message, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println("Failed to read the input:", err)
            continue
        }

        _, err = s.Write([]byte(message))
        if err != nil {
            panic(err)
        }
    }
}

func readMessage(s network.Stream) {
    r := bufio.NewReader(s)
    for {
        message, err := r.ReadString('\n')
        if err != nil {
            log.Println(err)
            return
        }

        fmt.Printf("From %s\n> %s", s.ID(), message)
    }
}

func handleDiscoveredPeer(host host.Host, peerInfo peer.AddrInfo) {
    if peerInfo.ID < host.ID() {
        if err := host.Connect(context.Background(), peerInfo); err != nil {
            panic(err)
        }
        fmt.Println("Connected to", peerInfo.String())
        
        s, err := host.NewStream(context.Background(), peerInfo.ID, ProtocolID)
        if err != nil {
            panic(err)
        }
        
        go writeMessage(s)
        go readMessage(s)
    }
}
func main() {
    cfg := getConfig()
    host, err := libp2p.New(libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/0", cfg.listenHost)))
    if err != nil {
        panic(err)
    }
    defer host.Close()

    // Print this node's addresses and ID
    fmt.Println("Addresses:", host.Addrs())
    fmt.Println("ID:", host.ID())
    fmt.Println("ProtocolID:", cfg.ProtocolID)

    // This gets called every time a peer connects and opens a stream to this node.
    host.SetStreamHandler(protocol.ID(cfg.ProtocolID), func(s network.Stream) {
        fmt.Println("new connection")
        handleStream(s)
        go writeMessage(s)
        go readMessage(s)
    })

    peerChan := initMDNS(host, "meet")
	for { // allows multiple peers to join
		peer := <-peerChan // will block until we discover a peer
		if peer.ID > host.ID() {
			// if other end peer id greater than us, don't connect to it, just wait for it to connect us
			fmt.Println("Found peer:", peer, " id is greater than us, wait for it to connect to us")
			continue
		}
        handleDiscoveredPeer(host, peer)
	}
}
