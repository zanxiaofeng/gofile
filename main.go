package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	http "github.com/siadat/gofile/http"
	log "github.com/siadat/gofile/log"
)

const usage = `Usage: gofile port [dir] [-v]`

var optRoot = ""

func main() {
	if verbose := flag.Bool("v", false, "enable verbose output"); *verbose {
		log.SetVerbose()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		log.Error(usage)
		os.Exit(1)
	}

	port, err := strconv.Atoi(flag.Args()[0])
	if err != nil {
		log.Error("Bad port number:", flag.Args()[0])
		os.Exit(1)
	}

	if flag.NArg() > 2 {
		log.Error("Too many args")
		os.Exit(1)
	}

	if flag.NArg() > 1 {
		optRoot = flag.Args()[1]
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%d", port))
	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Error(err)
		os.Exit(1)
		return
	}

	server := http.Server{Handler: fileServerHandleRequestGen(optRoot)}

	log.Normal("Starting server on port", port)
	server.Serve(ln)
}
