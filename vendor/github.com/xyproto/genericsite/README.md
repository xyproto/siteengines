# genericsite [![Build Status](https://travis-ci.com/xyproto/genericsite.svg?branch=master)](https://travis-ci.com/xyproto/genericsite) [![GoDoc](https://godoc.org/github.com/xyproto/genericsite?status.svg)](http://godoc.org/github.com/xyproto/genericsite)

Deprecated!
-----------

I'm phasing out this package.

* I want use [templates](https://github.com/unrolled/render) instead of generating webpages.
* I want to use the [permissions2](https://github.com/xyproto/permissions2) middleware for users and permissions.
* I don't want to use [web.go](https://github.com/hoisie/web), but rather keep the choice open and/or use the standard http package.

Features and limitations
------------------------

* Generic website library with search and user administration.
* Part of an experiment in procedural generation of websites.
* Uses procedural generation of html and css extensively.
* Has a user login and registration system.
* Has an email confirmation system.
* Has an admin panel.
* Supports subpages.
* Runs at native speed.
