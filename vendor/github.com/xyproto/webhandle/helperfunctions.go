package webhandle

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
)

// Returns a form parameter, or an empty string
func GetFormParam(req *http.Request, key string) string {
	return req.PostFormValue(key)
}

// Returns an url parameter, or an empty string
func GetParam(req *http.Request, key string) string {
	values, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		return ""
	}
	val := values.Get(key)
	return val
}

// Write the given string to the response writer. Convenience function.
func Ret(w http.ResponseWriter, s string) {
	fmt.Fprint(w, s)
}

// Get a value from an url.
// /hi/there/asdf with pos 2 returns asdf
func GetVal(url *url.URL, pos int) string {
	p := html.EscapeString(url.Path)
	fields := strings.Split(p, "/")
	if len(fields) <= pos {
		return ""
	}
	return fields[pos]
}

// Get the last value from an url.
// /hi/there/qwerty returns qwerty
func GetLast(url *url.URL) string {
	p := html.EscapeString(url.Path)
	fields := strings.Split(p, "/")
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

// Converts "true" or "false" to a bool
func TruthValue(val string) bool {
	return "true" == val
}

// Split a string into two strings at the colon
// If there's no colon, return the string and an empty string
func HostPortSplit(s string) (string, string) {
	if strings.Contains(s, ":") {
		sl := strings.SplitN(s, ":", 2)
		return sl[0], sl[1]
	}
	return s, ""
}

func TableCell(b bool) string {
	if b {
		return "<td class=\"yes\">yes</td>"
	}
	return "<td class=\"no\">no</td>"
}

func CleanUserInput(val string) string {
	return strings.Replace(val, "<", "&lt;", -1)
}
