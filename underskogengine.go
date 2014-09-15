package siteengines

// This "engine" is just started and is not complete

import (
	"net/http"

	. "github.com/xyproto/genericsite"
	"github.com/xyproto/permissions"
	"github.com/xyproto/simpleredis"
	. "github.com/xyproto/webhandle"
)

type UnderskogEngine struct {
	//plans *simpleredis.HashMap

	pool  *simpleredis.ConnectionPool // A connection pool for Redis
	state *permissions.UserState
}

func NewUnderskogEngine(state *permissions.UserState) *UnderskogEngine {
	return &UnderskogEngine{state.GetPool(), state}
}

func (ue *UnderskogEngine) ServePages(mux *http.ServeMux, basecp BaseCP, menuEntries MenuEntries) {
	underskogCP := basecp(ue.state)

	underskogCP.ContentTitle = "Mosebark"
	underskogCP.ExtraCSSurls = append(underskogCP.ExtraCSSurls, "/css/mosebark.css")

	tvgf := DynamicMenuFactoryGenerator(menuEntries)
	tvg := tvgf(ue.state)

	mux.HandleFunc("/mosebark", underskogCP.WrapHandle(mux, ue.GenerateMessages(), tvg))
	mux.HandleFunc("/css/mosebark.css", ue.GenerateCSS(underskogCP.ColorScheme))
}

func (ue *UnderskogEngine) GenerateMessages() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		//userdate := GetLast(req.URL)
		retval := ""
		retval += "<h1>MESSAGES</h1>"
		retval += BackButton()
		Ret(w, retval)
	}
}

func (ue *UnderskogEngine) GenerateCSS(cs *ColorScheme) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/css")
		Ret(w, `
.even {
	background-color: "a0a0a0;
}
.odd {
	background-color: #f0f0f0;
}
.yes {
	background-color: #90ff90;
	color: black;
}
.no {
	background-color: #ff9090;
	color: black;
}
table {
	border-collapse: collapse;
	padding: 1em;
	margin-top: 1.5em;
	margin-bottom: 1em;
}
table, th, tr, td {
	border: 1px solid black;
	padding: 1em;
}

.username:link { color: green; }
.username:visited { color: green; }
.username:hover { color: green; }
.username:active { color: green; }

.whitebg {
	background-color: white;
}

.darkgrey:link { color: #404040; }
.darkgrey:visited { color: #404040; }
.darkgrey:hover { color: #404040; }
.darkgrey:active { color: #404040; }

.somewhatcareful:link { color: #e09000; }
.somewhatcareful:visited { color: #e09000; }
.somewhatcareful:hover { color: #e09000; }
.somewhatcareful:active { color: #e09000; }

.careful:link { color: #e00000; }
.careful:visited { color: #e00000; }
.careful:hover { color: #e00000; }
.careful:active { color: #e00000; }

`)
		//
	}
}
