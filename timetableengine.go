package siteengines

import (
	"strconv"
	"strings"
	"time"

	"github.com/hoisie/web"
	"github.com/xyproto/calendar"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/personplan"
	"github.com/xyproto/pinterface"
	. "github.com/xyproto/webhandle"
)

// TODO: Simple font-symbol buttons for browsing backwards and forwards a day (or week): << < > >>

// TODO: Rename this module to something more generic than TimeTable
// TODO: Use the personplan and calendar module
// TODO: Add the timeTable pages to the search engine somehow (and the other engines too, like the chat)

/* Structure (TODO: Look at personplan for how the structure ended up)
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

type TimeTableEngine struct {
	state          pinterface.IUserState
	timeTableState *TimeTableState
}

type TimeTableState struct {
	// TODO: Find out how you are going to store the plans

	plans pinterface.IHashMap
}

func NewTimeTableEngine(userState pinterface.IUserState) (*TimeTableEngine, error) {

	creator := userState.Creator()

	timeTableState := new(TimeTableState)
	if plansHashMap, err := creator.NewHashMap("plans"); err != nil {
		return nil, err
	} else {
		timeTableState.plans = plansHashMap
		return &TimeTableEngine{userState, timeTableState}, nil
	}
}

func (tte *TimeTableEngine) ServePages(basecp BaseCP, menuEntries MenuEntries) {
	timeTableCP := basecp(tte.state)

	timeTableCP.ContentTitle = "TimeTable"
	timeTableCP.ExtraCSSurls = append(timeTableCP.ExtraCSSurls, "/css/timetable.css")

	tvgf := DynamicMenuFactoryGenerator(menuEntries)
	tvg := tvgf(tte.state)

	web.Get("/timetable", tte.GenerateTimeTableRedirect())                                  // Redirect to /timeTable/main
	web.Get("/timetable/(.*)", timeTableCP.WrapWebHandle(tte.GenerateShowTimeTable(), tvg)) // Displaying timeTable pages
	web.Get("/css/timetable.css", tte.GenerateCSS(timeTableCP.ColorScheme))                 // CSS that is specific for timeTable pages
}

func AllPlansDummyContent() *personplan.Plans {
	ppAlexander := personplan.NewPersonPlan("Alexander")
	ppAlexander.AddWorkday(time.Monday, 8, 15, "KNH")     // monday, from 8, up to 15
	ppAlexander.AddWorkday(time.Wednesday, 12, 17, "KOH") // wednesday, from 12, up to 17

	ppBob := personplan.NewPersonPlan("Bob")
	ppBob.AddWorkday(time.Monday, 9, 11, "KOH")   // monday, from 9, up to 11
	ppBob.AddWorkday(time.Thursday, 8, 10, "KNH") // wednesday, from 8, up to 10

	periodplan := personplan.NewSemesterPlan(2013, 1, 8)
	periodplan.AddPersonPlan(ppAlexander)
	periodplan.AddPersonPlan(ppBob)

	allPlans := personplan.NewPlans()
	allPlans.AddSemesterPlan(periodplan)

	return allPlans
}

func RenderWeekFrom(t time.Time, locale string) string {

	allPlans := AllPlansDummyContent()

	cal, err := calendar.NewCalendar(locale, true)
	if err != nil {
		panic("Could not create a calendar for locale " + locale + "!")
	}

	retval := ""
	retval += "<table>"

	// Headers
	retval += "<tr>"
	retval += "<td></td>"

	// Loop through 7 days from the given date
	current := t
	for i := 0; i < 7; i++ {

		// Cell
		retval += "<td><b>"

		// Contents
		retval += Num2dd(current.Day()) + ". " + cal.MonthName(current.Month())

		// End of cell
		retval += "</b></td>"

		// Advance to the next day
		current = current.AddDate(0, 0, 1)
	}

	// End of headers
	retval += "</tr>"

	// Each row is an hour
	for hour := 8; hour < 22; hour++ {
		retval += "<tr>"

		// Each column is a day
		retval += "<td>kl. " + Num2dd(hour) + ":00</td>"

		// Loop through 7 days from the given date, while using the correct hour
		current := time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, time.UTC)

		for i := 0; i < 7; i++ {

			// Cell with contents
			red, desc, _ := cal.RedDay(current)
			if red {
				retval += "<td bgcolor='#ffb0b0'>" + desc + "</td>"
			} else {
				retval += "<td>" + allPlans.HTMLHourEvents(current) + "</td>"
			}

			// Advance to the next day
			current = current.AddDate(0, 0, 1)
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

func (we *TimeTableEngine) GenerateShowTimeTable() WebHandle {
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

		weekstart := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

		retval += RenderWeekFrom(weekstart, "nb_NO")
		retval += BackButton()
		return retval
	}
}

func (we *TimeTableEngine) GenerateTimeTableRedirect() SimpleContextHandle {
	return func(ctx *web.Context) string {
		t := time.Now()
		// Redirect to the current date on the form yyyy-mm-dd
		ctx.SetHeader("Refresh", "0; url=/timetable/"+t.String()[:10], true)
		return ""
	}
}

func (tte *TimeTableEngine) GenerateCSS(cs *ColorScheme) SimpleContextHandle {
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
