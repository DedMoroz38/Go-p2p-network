package main

type config struct {
	ProtocolID			 string
	listenHost       string
	listenPort       string
}

func getConfig() *config{
	c := &config{}

	c.ProtocolID = "/chat/1.0.0"
	c.listenHost = "127.0.0.1"
	c.listenPort = "0"

	return c
}