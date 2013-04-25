package siteengines

import (
	"time"

	. "github.com/xyproto/browserspeak"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/simpleredis"
	"github.com/xyproto/web"
)

// TODO: Add the ftls pages to the search engine somehow (and the other engines too, like the chat)

const (
	SPRING = iota
	SUMMER
	AUTUMN
)

type FTLSEngine struct {
	userState *UserState
	ftlsState *FTLSState
}

type TimeRange struct {
	id                    int
	date                  time.Time
	fromHourNumber        int
	toHourNumberInclusive int
}

type Vakt struct {
	id       int
	username string
	when     TimeRange
}

type VaktDay struct {
	id              int
	dayOfTheWeek    int
	fromHour        int
	toHourInclusive int
}

type VaktPerson struct {
	id         int
	username   string
	vaktDayIds []int
}

type VaktPlan struct {
	id            int
	year          int
	period        int // Period constant
	vaktPersonIds []int
}

type FTLSState struct {
	// FTLS/vakt related
	timeRanges *simpleredis.HashMap
	vakt       *simpleredis.HashMap
	vaktDay    *simpleredis.HashMap
	vaktPerson *simpleredis.HashMap
	vaktPlan   *simpleredis.HashMap

	// Which data is really stored for FTLS?
	pool *simpleredis.ConnectionPool // A connection pool for Redis
}

func NewFTLSEngine(userState *UserState) *FTLSEngine {
	pool := userState.GetPool()
	ftlsState := new(FTLSState)

	// FTLS/vakt related
	ftlsState.timeRanges = simpleredis.NewHashMap(pool, "ftlsTimeRanges")
	ftlsState.vakt = simpleredis.NewHashMap(pool, "ftlsVakt")
	ftlsState.vaktDay = simpleredis.NewHashMap(pool, "ftlsVaktDay")
	ftlsState.vaktPerson = simpleredis.NewHashMap(pool, "ftlsVaktPerson")
	ftlsState.vaktPlan = simpleredis.NewHashMap(pool, "ftlsVaktPlan")

	ftlsState.pool = pool
	return &FTLSEngine{userState, ftlsState}
}

func (we *FTLSEngine) ServePages(basecp BaseCP, menuEntries MenuEntries) {
	ftlsCP := basecp(we.userState)
	ftlsCP.ContentTitle = "FTLS"
	ftlsCP.ExtraCSSurls = append(ftlsCP.ExtraCSSurls, "/css/ftls.css")

	tvgf := DynamicMenuFactoryGenerator(menuEntries)
	tvg := tvgf(we.userState)

	web.Get("/ftls", we.GenerateFTLSRedirect())                             // Redirect to /ftls/main
	web.Get("/ftls/(.*)", ftlsCP.WrapWebHandle(we.GenerateShowFTLS(), tvg)) // Displaying ftls pages
	web.Get("/css/ftls.css", we.GenerateCSS(ftlsCP.ColorScheme))            // CSS that is specific for ftls pages
}

func (we *FTLSEngine) GenerateShowFTLS() WebHandle {
	return func(ctx *web.Context, weeknr string) string {
		retval := ""
		retval += "<h1>Hi</h1>"
		retval += BackButton()
		return retval
	}
}

func (we *FTLSEngine) GenerateFTLSRedirect() SimpleContextHandle {
	return func(ctx *web.Context) string {
		// TODO: Redirect to the current week nr
		ctx.SetHeader("Refresh", "0; url=/ftls/0", true)
		return ""
	}
}

func (we *FTLSEngine) GenerateCSS(cs *ColorScheme) SimpleContextHandle {
	return func(ctx *web.Context) string {
		ctx.ContentType("css")
		return `
.yes {
	background-color: #90ff90;
	color: black;
}
.no {
	background-color: #ff9090;
	color: black;
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

#ftlsText {
	background-color: white;
}

`
		//
	}
}