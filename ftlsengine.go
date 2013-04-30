package siteengines

import (
	"strconv"
	"strings"
	"time"

	. "github.com/xyproto/browserspeak"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/simpleredis"
	"github.com/xyproto/web"
)

// TODO: Add the ftls pages to the search engine somehow (and the other engines too, like the chat)

type TimeRange struct {
	id                    int
	date                  time.Time
	fromHourNumber        int
	toHourNumberInclusive int
}

// Changes in the plan is what it's really about
type VaktChange struct {
	id        int
	username  string
	timeRange int
	toggleOn  bool
}

// Period
const (
	SPRING = iota
	SUMMER
	AUTUMN
)

type VaktPlan struct {
	id          int
	year        int
	period      int // SPRING, SUMMER or AUTUMN
	vaktChanges []int
}

type FTLSEngine struct {
	userState *UserState
	ftlsState *FTLSState
}

type FTLSState struct {
	// FTLS/vakt related
	timeRanges *simpleredis.HashMap
	vaktChange *simpleredis.HashMap
	vaktPlan   *simpleredis.HashMap

	// Which data is really stored for FTLS?
	pool *simpleredis.ConnectionPool // A connection pool for Redis
}

func NewFTLSEngine(userState *UserState) *FTLSEngine {
	pool := userState.GetPool()
	ftlsState := new(FTLSState)

	// FTLS/vakt related
	ftlsState.timeRanges = simpleredis.NewHashMap(pool, "ftlsTimeRanges")
	ftlsState.vaktChange = simpleredis.NewHashMap(pool, "ftlsVaktChange")
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

// TODO: Find a more clever way to translate everything
func MonthName(month int, language string) string {
	var names []string

	// TODO: Return err or "" instead of panic?
	if month == 0 {
		panic("Invalid month number: 0")
	} else if month >= 13 {
		panic("Month number too high: " + strconv.Itoa(month))
	}

	if language == "no" {
		names = []string{"NIL", "januar", "februar", "mars", "april", "mai", "juni", "juli", "august", "september", "oktober", "november", "desember"}
	} else {
		names = []string{"NIL", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	}

	return names[month]
}

func RenderWeekFrom(year, month, startday int) string {
	retval := ""
	retval += "<table>"

	// Headers
	retval += "<tr>"
	for day := startday; day < (startday + 7); day++ {
		retval += "<td><b>" + Num2dd(day) + ". " + MonthName(month, "no") + "</b></td>"
	}
	retval += "</tr>"

	// Each row is an hour
	for hour := 8; hour < 22; hour++ {
		retval += "<tr>"
		// TODO: Use time/date functions for adding days instead, these months can go to day 37...
		// Each column is a day
		for day := startday; day < (startday + 7); day++ {
			retval += "<td>kl. " + Num2dd(hour) + ":00</td>"
		}
		retval += "</tr>"
	}

	retval += "</table>"
	return retval
}

// Convert from a number to a double digit string
func Num2dd(num int) string {
	s := strconv.Itoa(num)
	if len(s) == 2 {
		return s
	}
	return "0" + s
}

func (we *FTLSEngine) GenerateShowFTLS() WebHandle {
	return func(ctx *web.Context, date string) string {
		ymd := strings.Split(date, "-")
		// TODO: Error checking when splitting and converting + user input cleanup
		year, _ := strconv.Atoi(ymd[0])
		month, _ := strconv.Atoi(ymd[1])
		day, _ := strconv.Atoi(ymd[2])
		retval := ""
		retval += "<h1>En uke fra " + strconv.Itoa(year) + "-" + Num2dd(month) + "-" + Num2dd(day) + "</h1>"
		retval += RenderWeekFrom(year, month, day)
		retval += BackButton()
		return retval
	}
}

func (we *FTLSEngine) GenerateFTLSRedirect() SimpleContextHandle {
	return func(ctx *web.Context) string {
		t := time.Now()
		// Redirect to the current date on the form yyyy-mm-dd
		ctx.SetHeader("Refresh", "0; url=/ftls/"+t.String()[:10], true)
		return ""
	}
}

func (we *FTLSEngine) GenerateCSS(cs *ColorScheme) SimpleContextHandle {
	return func(ctx *web.Context) string {
		ctx.ContentType("css")
		return `
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

`
		//
	}
}
