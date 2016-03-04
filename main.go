package main

import (
	"github.com/docopt/docopt-go"
	http "github.com/siadat/gofile/http"
	log "github.com/siadat/gofile/log"
)

const usage = `Usage: gofile [-v] <port>`

var (
	optPort    = "8080"
	optVerbose = false
)

func main() {
	args, _ := docopt.Parse(usage, nil, true, http.Version, false, true)
	if args["-v"].(bool) {
		log.Level = log.LevelVerbose
	}
	log.Normal("Starting server on port", args["<port>"].(string))
	http.Serve(args["<port>"].(string), fileServerHandleRequest)
}
