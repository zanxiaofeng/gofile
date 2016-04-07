host=127.0.0.1
port=8080

CRLF="\r\n"
HTTP11="HTTP/1.1${CRLF}"
HConn="Connection: close${CRLF}"
HHost="Host: localhost${CRLF}"
HIfModifiedPast="If-Modified-Since: $(gdate +"%a, %d %b %Y %T %Z" --date='@1000000000')${CRLF}"
HIfModifiedFuture="If-Modified-Since: $(gdate +"%a, %d %b %Y %T %Z" --date='@9000000000')${CRLF}"
HByteRange1="Range: bytes=-10${CRLF}"
HByteRange2="Range: bytes=90-${CRLF}"
HByteRange3="Range: bytes=10-10${CRLF}"
HByteRange4="Range: bytes=10-20${CRLF}"
HByteRange5="Range: bytes=-1${CRLF}"
HByteRange6="Range: bytes=0-10,20-${CRLF}" # Not supported

sendreq() {
	local ColorReq="\e[94m"
	local ColorNon="\e[0m"
	echo # "------------------------------------"
	echo -ne "${ColorReq}"
	echo -e "$1"
	echo -ne "${ColorNon}"
	echo -e "$1" | nc $host $port | head -n 15
}

hl() {
	echo
	echo "===================================="
	echo "$1"
	echo "===================================="
}

hl "Valid: relative url => 200"
sendreq "GET / ${HTTP11}${HHost}${HConn}"
sendreq "HEAD / ${HTTP11}${HHost}${HConn}"

hl "Valid: modified, normal response => 200"
sendreq "GET /testdata/date.txt ${HTTP11}${HHost}${HIfModifiedPast}${HConn}"

hl "Valid: not modified, body should be empty => 304"
sendreq "GET /testdata/date.txt ${HTTP11}${HHost}${HIfModifiedFuture}${HConn}"

hl "Valid: range header => 200"
sendreq "GET /testdata/100.txt ${HTTP11}${HHost}${HByteRange1}${HConn}"
sendreq "GET /testdata/100.txt ${HTTP11}${HHost}${HByteRange2}${HConn}"
sendreq "GET /testdata/100.txt ${HTTP11}${HHost}${HByteRange3}${HConn}"
sendreq "GET /testdata/100.txt ${HTTP11}${HHost}${HByteRange4}${HConn}"
sendreq "GET /testdata/100.txt ${HTTP11}${HHost}${HByteRange5}${HConn}"
sendreq "GET /testdata/100.txt ${HTTP11}${HHost}${HByteRange6}${HConn}"

hl "Valid: absolute url => 200"
sendreq "GET http://localhost:8080/ ${HTTP11}${HHost}${HConn}"

hl "Valid: should not be chunked, no Transfer-Encoding header, must have Content-Length => 200"
sendreq "GET /testdata/date.txt ${HTTP11}${HHost}${HConn}"

hl "Valid: special characters => 200"
sendreq "GET /testdata/a+b+c+(d)/یک.txt ${HTTP11}${HHost}${HConn}"
sendreq "GET /testdata/a+b+c+(d)/e+f+g+[h]/test.txt ${HTTP11}${HHost}${HConn}"

hl "Valid: should not found => 404"
sendreq "GET /foo ${HTTP11}${HHost}${HConn}"

hl "Invalid: no 'Host' header => 400"
sendreq "GET / ${HTTP11}${HConn}"

hl "Invalid: bad paths => 401"
sendreq "GET ../ ${HTTP11}${HHost}${HConn}"
sendreq "GET /.. ${HTTP11}${HHost}${HConn}"
sendreq "GET http://localhost:8080/../ ${HTTP11}${HHost}${HConn}"

hl "Invalid: bad method => 501"
sendreq "POST / ${HTTP11}${HHost}${HConn}"

hl "A file larger than the response chunk size (1M) should be identical to the original file"
largefile="2m.file"
dd if=/dev/zero of=testdata/"$largefile" bs=2048 count=1024
wget -O "$largefile" "http://${host}:${port}/testdata/${largefile}"
diff -s "$largefile" testdata/"$largefile"
rm "$largefile" testdata/"$largefile"

hl "Valid: keepalive should not disconnect, because no '${HConn}' header is present"
sendreq "GET / ${HTTP11}${HHost}"
