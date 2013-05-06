package siteengines

import (
	"strconv"
	"strings"
	"time"

	"github.com/hoisie/web"
	. "github.com/xyproto/browserspeak"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/simpleredis"
)

// TODO: Add the ftls pages to the search engine somehow (and the other engines too, like the chat)

/* Structure
 *
 * Three layers:
 *  workdays
 *  peopleplans
 *  hourchanges
 *
 * The workdays are automatically generated, no input needed.
 * A PeoplePlan is which hours, which days, from when to when a person is going to work
 * An HourChange is a change for a specific hour, from a username (if any), to a username
 *
 * There should exists functions that:
 * Can tell which hours a person actually ended up owning, after changes
 * Can tell how a day will look, after changes
 *
 */

type FTLSEngine struct {
	userState *UserState
	ftlsState *FTLSState
}

type FTLSState struct {
	workdays    *simpleredis.HashMap
	peopleplans *simpleredis.HashMap
	hourchanges *simpleredis.HashMap

	// Which data is really stored for FTLS?
	pool *simpleredis.ConnectionPool // A connection pool for Redis
}

func NewFTLSEngine(userState *UserState) *FTLSEngine {
	pool := userState.GetPool()
	ftlsState := new(FTLSState)

	ftlsState.workdays = simpleredis.NewHashMap(pool, "workdays")
	ftlsState.peopleplans = simpleredis.NewHashMap(pool, "peopleplans")
	ftlsState.hourchanges = simpleredis.NewHashMap(pool, "hourchanges")

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

// TODO: Find a more clever system to translate everything
func MonthName(month int, language string) string {
	var names []string

	if month <= 0 {
		//panic("Invalid month number: 0")
		return "NIL"
	} else if month >= 13 {
		//panic("Month number too high: " + strconv.Itoa(month))
		return "NIL"
	}

	if language == "no" {
		names = []string{"januar", "februar", "mars", "april", "mai", "juni", "juli", "august", "september", "oktober", "november", "desember"}
	} else {
		names = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	}

	return names[month-1]
}

func RenderWeekFrom(year, month, startday int) string {
	retval := ""
	retval += "<table>"

	// TODO: Convert year/month/startday back to Time, but in a safe way. Look at the parse function for time.

	// Headers
	retval += "<tr>"
	retval += "<td></td>"
	for day := startday; day < (startday + 7); day++ {
		retval += "<td><b>" + Num2dd(day) + ". " + MonthName(month, "no") + "</b></td>"
	}
	retval += "</tr>"

	// Each row is an hour
	for hour := 8; hour < 22; hour++ {
		retval += "<tr>"
		// TODO: Use time/date functions for adding days instead, these months can go to day 37...
		// Each column is a day
		retval += "<td>kl. " + Num2dd(hour) + ":00</td>"
		for day := startday; day < (startday + 7); day++ {
			retval += "<td>FREE</td>"
		}
		retval += "</tr>"
	}

	retval += "</table>"
	return retval
}

// Convert from a number to a double digit string
func Num2dd(num int) string {
	s := strconv.Itoa(num)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func (we *FTLSEngine) GenerateShowFTLS() WebHandle {
	return func(ctx *web.Context, userdate string) string {
		date := CleanUserInput(userdate)
		ymd := strings.Split(date, "-")
		if len(ymd) != 3 {
			return "Invalid yyyy-mm-dd: " + date
		}
		year, err := strconv.Atoi(ymd[0])
		if (err != nil) || (len(ymd[0]) != 4) {
			return "Invalid year: " + ymd[0]
		}
		month, err := strconv.Atoi(ymd[1])
		if (err != nil) || (len(ymd[1]) > 2) {
			return "Invalid month: " + ymd[1]
		}
		day, err := strconv.Atoi(ymd[2])
		if (err != nil) || (len(ymd[2]) > 2) {
			return "Invalid day: " + ymd[2]
		}
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
