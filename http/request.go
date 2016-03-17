package http

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	Method       string
	URL          *url.URL
	UnescapedURL string
	protocol     string
	Headers      map[string]string
	LocalAddr    net.Addr
	Ranges       []ByteRange
	RangedReq    bool
}

var (
	reAbsURL = regexp.MustCompile("^https?://[^/]+")
)

type ByteRange struct {
	Start int64
	End   int64
}

func (br ByteRange) Length() int64 {
	return br.End - br.Start + 1
}

func parseByteRangeHeader(headerValue string) (byteRanges []ByteRange, explicit bool) {
	rangePrefix := "bytes="
	byteRanges = make([]ByteRange, 0)

	if !strings.HasPrefix(headerValue, rangePrefix) {
		byteRanges = append(byteRanges, ByteRange{Start: 0, End: -1})
		return
	}

	explicit = true

	headerValue = headerValue[len(rangePrefix):]
	// regexp.MustCompile(`^(-\d+|\d+-|\d+-\d+)$`)
	for _, value := range strings.Split(headerValue, ",") {

		// Let's say we have 10 bytes: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
		// -10 => [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
		// -7 => [3, 4, 5, 6, 7, 8, 9]
		// -2 => [8, 9]
		// -1 => [9]
		if val, err := strconv.ParseInt(value, 10, 0); err == nil && val < 0 {
			byteRanges = append(byteRanges, ByteRange{Start: val, End: -1})
			continue
		}

		// 0- => [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
		// 3- => [3, 4, 5, 6, 7, 8, 9]
		if strings.HasSuffix(value, "-") {
			if val, err := strconv.ParseInt(value[:len(value)-1], 10, 0); err == nil {
				byteRanges = append(byteRanges, ByteRange{Start: val, End: -1})
			}
			continue
		}

		// 1-1 => [1, 1]
		// 3-6 => [3, 4, 5, 6]
		rangeVals := strings.Split(value, "-")
		val1, err1 := strconv.ParseInt(rangeVals[0], 10, 0)
		val2, err2 := strconv.ParseInt(rangeVals[1], 10, 0)
		if err1 == nil && err2 == nil {
			byteRanges = append(byteRanges, ByteRange{Start: val1, End: val2})
		}
	}
	return
}

func ParseDate(date string) (t time.Time) {
	t, err := time.Parse(HTTPTimeFormat, date)
	if err != nil {
		fmt.Println("error parsing", err)
	}
	return
}

func (req *Request) ParseInitialLine(line string) (err error) {
	words := strings.SplitN(line, " ", 3)
	if len(words) != 3 {
		err = errors.New("Invalid initial request line.")
		return
	}
	req.Method = words[0]
	req.URL, _ = url.Parse(words[1])
	req.protocol = words[2]

	return
}

func (req *Request) ParseHeaders(headerLines []string) {
	for _, headerLine := range headerLines {
		headerPair := strings.SplitN(headerLine, ": ", 2)
		if len(headerPair) == 2 {
			req.Headers[headerPair[0]] = headerPair[1]
		}
	}
	req.Ranges, req.RangedReq = parseByteRangeHeader(req.Headers["Range"])
}
