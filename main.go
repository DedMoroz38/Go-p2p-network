package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"

	// "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

const protocolID = "/example/1.0.0"
// const discoveryNamespace = "example"

var streams []network.Stream

func main() {
    // Add -peer-address flag
    peerAddr := flag.String("peer-address", "", "peer address")
    flag.Parse()

    // Create the libp2p host.
    //
    // Note that we are explicitly passing the listen address and restricting it to IPv4 over the
    // loopback interface (127.0.0.1).
    //
    // Setting the TCP port as 0 makes libp2p choose an available port for us.
    // You could, of course, specify one if you like.
    host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
    if err != nil {
        panic(err)
    }
    defer host.Close()

    // Print this node's addresses and ID
    fmt.Println("Addresses:", host.Addrs())
    fmt.Println("ID:", host.ID())
    fmt.Print("command to connect: ./p2p -peer-address ", host.Addrs()[0], "/p2p/", host.ID())

    // Setup a stream handler.
    //
    // This gets called every time a peer connects and opens a stream to this node.
    host.SetStreamHandler(protocolID, func(s network.Stream) {
        fmt.Println("new connection")
        handleStream(s)
        go writeMessage(s)
        go readMessage(s)
    })

    notifee := &discoveryNotifee{}

    // Setup peer discovery.
    discoveryService := mdns.NewMdnsService(
        host,
        "discoverer",
        notifee,
    )
    if err != nil {
        panic(err)
    }
    defer discoveryService.Close()

    discoveryService.Start()

    // If we received a peer address, we should connect to it.
    if *peerAddr != "" {
        // Parse the multiaddr string.
        peerMA, err := multiaddr.NewMultiaddr(*peerAddr)
        if err != nil {
            panic(err)
        }
        peerAddrInfo, err := peer.AddrInfoFromP2pAddr(peerMA)
        if err != nil {
            panic(err)
        }

        // Connect to the node at the given address.
        if err := host.Connect(context.Background(), *peerAddrInfo); err != nil {
            panic(err)
        }
        fmt.Println("Connected to", peerAddrInfo.String())

        // Open a stream with the given peer.
        s, err := host.NewStream(context.Background(), peerAddrInfo.ID, protocolID)
        if err != nil {
            panic(err)
        }

        // Start the write and read threads.
        go writeMessage(s)
        go readMessage(s)
    }

    sigCh := make(chan os.Signal)
    signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
    <-sigCh
}

// used to append the stream to the list of streams to pass recieved messages to all connected streams
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

        // Write the received message to all streams
        for _, stream := range streams {
            _, err := stream.Write([]byte(message))
            if err != nil {
                log.Println(err)
            }
        }
    }
}

type discoveryNotifee struct {
    h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
    fmt.Println("found peer", peerInfo.String())
}