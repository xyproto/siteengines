#webhandle [![Build Status](https://travis-ci.org/xyproto/webhandle.svg?branch=master)](https://travis-ci.org/xyproto/webhandle) [![GoDoc](https://godoc.org/github.com/xyproto/webhandle?status.svg)](http://godoc.org/github.com/xyproto/webhandle)

One way to serve webpages with [onthefly](https://github.com/xyproto/onthefly) and [mux](https://github.com/gorilla/mux).

This is an experimental package and the overall design could be cleaner. The entire thing should be rewritten.

Online API Documentation
------------------------

[godoc.org](http://godoc.org/github.com/xyproto/webhandle)

Features and limitations
------------------------

* Webhandle can take a `*onthefly.Page` and publish both the HTML and CSS together, by listening to HTTP GET requests.
* `gorilla/mux` is used for some of the functions.

General information
-------------------

* Version: 0.1.1
* License: MIT
* Alexander F. Rødseth

