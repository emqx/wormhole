package main

import (
	"fmt"
	"github.com/emqx/wormhole/client"
	"github.com/emqx/wormhole/server"
	"os"
	"strings"
)

func main() {
	if args := os.Args[1:]; len(args) == 2 {
		if mode := strings.ToLower(args[0]); mode == "client" {
			client.NewClient()
		} else {
			fmt.Println("Invalid argument, expect 'wormhole client d62ef200-4e59-11eb-9890-f45c89b00d3d`.")
			return
		}
	} else if len(args) == 0 {
		server.NewServer()
	}
}
