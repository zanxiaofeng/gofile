CRLF="\r\n"
host=127.0.0.1
port=8080
HTTP11="HTTP/1.1${CRLF}"
HConn="Connection: close${CRLF}"
HHost="Host: localhost${CRLF}"
HIfModifiedPast="If-Modified-Since: $(gdate +"%a, %d %b %Y %T %Z" --date='@1000000000')${CRLF}"
HIfModifiedFuture="If-Modified-Since: $(gdate +"%a, %d %b %Y %T %Z" --date='@9000000000')${CRLF}"
echo $HIfModifiedPast
echo $HIfModifiedFuture

sendreq() {
	echo "------------------------------------"
	echo -e "$1"
	echo -e "$1" | nc $host $port | head -n 10
}

# Valid: relative url => 200
sendreq "GET / ${HTTP11}${HHost}${HConn}"
sendreq "HEAD / ${HTTP11}${HHost}${HConn}"

# Valid: modified, normal response => 200
sendreq "GET /testdata/date.txt ${HTTP11}${HHost}${HIfModifiedPast}${HConn}"

# Valid: not modified, body should be empty => 304
sendreq "GET /testdata/date.txt ${HTTP11}${HHost}${HIfModifiedFuture}${HConn}"

# Valid: absolute url => 200
sendreq "GET http://localhost:8080/ ${HTTP11}${HHost}${HConn}"

# Valid: should not be chunked, no Transfer-Encoding header, must have Content-Length => 200
sendreq "GET /testdata/date.txt ${HTTP11}${HHost}${HConn}"

# Valid: should not found => 404
sendreq "GET /foo ${HTTP11}${HHost}${HConn}"

# Invalid: no 'Host' header => 400
sendreq "GET / ${HTTP11}${HConn}"

# Invalid: bad paths => 401
sendreq "GET ../ ${HTTP11}${HHost}${HConn}"
sendreq "GET /.. ${HTTP11}${HHost}${HConn}"
sendreq "GET http://localhost:8080/../ ${HTTP11}${HHost}${HConn}"

# Invalid: bad method => 501
sendreq "POST / ${HTTP11}${HHost}${HConn}"

# Valid: keepalive should not disconnect, because no '${HConn}' header is present
sendreq "GET / ${HTTP11}${HHost}"
