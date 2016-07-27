package http

import (
	"fmt"
	"testing"
)

// func TestServe() { }

func ExampleServe(t *testing.T) {
	Serve(8080, func(req Request, res *Response) {
		defer close(res.Body)
		res.Body <- []byte(fmt.Sprintf("You requested %s", req.URL))
	})
}
