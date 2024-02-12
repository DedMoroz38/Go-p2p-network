package main

import (
	"bufio"
	"context"
	"math/rand"
	"time"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const protocolID = "/chat/1.0.0"
// const discoveryNamespace = "example"

var streams []network.Stream

func main() {
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

    notifee := &discoveryNotifee{host}

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

    sigCh := make(chan os.Signal)
    signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
    <-sigCh
}

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

type discoveryNotifee struct {
    h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
    rand.Seed(time.Now().UnixNano())
    randomDuration := time.Duration(rand.Intn(3)) * time.Second

    // Wait for the random duration
    time.Sleep(randomDuration)
    if err := n.h.Connect(context.Background(), peerInfo); err != nil {
        panic(err)
    }
    fmt.Println("Connected to", peerInfo.String())

    s, err := n.h.NewStream(context.Background(), peerInfo.ID, protocolID)
    if err != nil {
        panic(err)
    }

    go writeMessage(s)
    go readMessage(s)
}