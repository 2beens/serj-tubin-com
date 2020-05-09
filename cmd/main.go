package main

import (
	"fmt"

	"github.com/2beens/serjtubincom/internal"
)

func main() {
	fmt.Println("starting ...")
	server := internal.NewServer()
	server.Serve()
}
