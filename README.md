# Gofile

A non-blocking directory listing and file server.
It implementats HTTP/1.1 keepalive, chunked transfer, and byte range.

This tool is built for learning purpose only. It is not intended to be used in production.

![gofile](/../screenshots/screenshot-0.1.0.png?raw=true "gofile")

### Usage

    Usage: gofile [-v] <port> [<root>]

Examples:

    gofile 8080
    gofile 8080 ~/public

### Install

    go get -u github.com/siadat/gofile

### HTTP/1.1 implementation checklist

- [x] GET and HEAD methods
- [x] Support keep-alive connections
- [x] Support chunked transfer encoding
- [x] Requests must include a `Host` header
- [x] Requests with `Connection: close` should be closed
- [x] Support for requests with absolute URLs
- [x] If-Modified-Since support
- [x] Byte range support
- [ ] Transparent response compression
- [ ] Optimize for speed
