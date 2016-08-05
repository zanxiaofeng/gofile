package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"

	http "github.com/siadat/gofile/http"
)

const usage = `Usage: gofile port [dir]`

var optRoot = ""

func main() {
	flag.Usage = func() {
		log.Fatal(usage)
	}

	flag.Parse()

	if flag.NArg() < 1 || flag.NArg() > 2 {
		flag.Usage()
	}

	port, err := strconv.Atoi(flag.Args()[0])
	if err != nil {
		log.Fatal("Bad port number:", flag.Args()[0])
	}

	if flag.NArg() == 2 {
		optRoot = flag.Args()[1]
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	server := http.Server{Handler: fileServerHandleRequestGen(optRoot)}
	log.Println("Starting server on port", port)
	server.Serve(ln)
}
