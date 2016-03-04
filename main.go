package main

import (
	"github.com/docopt/docopt-go"
	http "github.com/siadat/gofile/http"
	log "github.com/siadat/gofile/log"
)

const usage = `Usage: gofile <port> [-v]`

var (
	optPort    = "8080"
	optVerbose = false
)

func main() {
	args, _ := docopt.Parse(usage, nil, true, "version 0.2.0", false, true)
	if args["-v"].(bool) {
		log.Level = log.LevelVerbose
	}
	log.Normal("Starting server on port", args["<port>"].(string))
	http.Serve(args["<port>"].(string), fileServerHandleRequest)
}
