package main

import (
	"github.com/docopt/docopt-go"
	http "github.com/siadat/gofile/http"
	log "github.com/siadat/gofile/log"
)

const usage = `Usage: gofile [-v] <port> [<root>]`

var (
	optPort    = "8080"
	optVerbose = false
	optRoot    = ""
)

func main() {
	args, _ := docopt.Parse(usage, nil, true, http.Version, false, true)
	if args["-v"].(bool) {
		log.SetVerbose()
	}
	log.Normal("Starting server on port", args["<port>"].(string))
	if args["<root>"] != nil {
		optRoot = args["<root>"].(string)
	}
	http.Serve(args["<port>"].(string), fileServerHandleRequestGen(optRoot))
}
