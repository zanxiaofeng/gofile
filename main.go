package main

import (
	"os"
	"strconv"

	"github.com/docopt/docopt-go"
	http "github.com/siadat/gofile/http"
	log "github.com/siadat/gofile/log"
)

const usage = `Usage: gofile [-v] <port> [<root>]`

var (
	optVerbose = false
	optRoot    = ""
)

func main() {
	args, _ := docopt.Parse(usage, nil, true, "0.3.0", false, true)
	if args["-v"].(bool) {
		log.SetVerbose()
	}
	log.Normal("Starting server on port", args["<port>"].(string))
	if args["<root>"] != nil {
		optRoot = args["<root>"].(string)
	}
	port, err := strconv.Atoi(args["<port>"].(string))
	if err != nil {
		log.Error("Bad port number")
		os.Exit(1)
	}
	http.Serve(port, fileServerHandleRequestGen(optRoot))
}
