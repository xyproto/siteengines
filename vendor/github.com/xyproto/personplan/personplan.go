package personplan

import (
	"fmt"
	"time"

	"github.com/xyproto/calendar"
)

const (
	FIRSTHOUR = 8
	LASTHOUR  = 22
)

// Info about one thing that can happen during an hour
type HourEvent struct {
	who   string
	when  time.Time
	where string
}

// Info about everything that happens during an hour, per person
type HourEvents []HourEvent

// A plan is a collecion of plans for just a few months at a time
type Plans struct {
	all []*SemesterPlan
}

type WorkDayAndLocation struct {
	dayoftheweek time.Weekday
	fromHour     int
	uptoHour     int
	location     string
}

type PersonPlan struct {
	who      string
	workdays []*WorkDayAndLocation
	locale   string
}

type SemesterPlan struct {
	year        int
	fromMonth   int
	uptoMonth   int
	personPlans []*PersonPlan
}

func NewPersonPlan(who string) *PersonPlan {
	var pp PersonPlan
	pp.who = who
	pp.locale = "nb_NO"
	return &pp
}

func (pp *PersonPlan) AddWorkday(dayoftheweek time.Weekday, fromHour, uptoHour int, location string) {
	newday := &WorkDayAndLocation{dayoftheweek, fromHour, uptoHour, location}
	pp.workdays = append(pp.workdays, newday)
}

func (pp *PersonPlan) String() string {
	cal, err := calendar.NewCalendar(pp.locale, true)
	if err != nil {
		panic("No calendar available for " + pp.locale)
	}
	s := "User: " + pp.who + "\n"
	s += "-----------------------------------------------\n"
	for _, day := range pp.workdays {
		s += "\n"
		s += "\t" + day.dayoftheweek.String() + " (" + cal.DayName(day.dayoftheweek) + ")\n"
		s += fmt.Sprintf("\tFrom this hour: \t%d\n", day.fromHour)
		s += fmt.Sprintf("\tUp to this hour:\t%d\n", day.uptoHour)
		s += fmt.Sprintf("\tAt this location:\t%s\n", day.location)
	}
	return s
}

func NewSemesterPlan(year, fromMonth, uptoMonth int) *SemesterPlan {
	var pps []*PersonPlan
	return &SemesterPlan{year, fromMonth, uptoMonth, pps}
}

func (sp *SemesterPlan) AddPersonPlan(persplan *PersonPlan) {
	sp.personPlans = append(sp.personPlans, persplan)
}

func (sp *SemesterPlan) ForAllWeekdays(fn func(string, time.Weekday, int, string) string) string {
	s := ""
	for day := 0; day < 7; day++ {
		for hour := FIRSTHOUR; hour <= LASTHOUR; hour++ {
			for _, persplan := range sp.personPlans {
				for _, personday := range persplan.workdays {
					if personday.dayoftheweek == time.Weekday(day) {
						if (hour >= personday.fromHour) && (hour < personday.uptoHour) {
							s += fn(persplan.who, time.Weekday(day), hour, personday.location)
						}
					}
				}
			}
		}
	}
	return s
}

func infoline(who string, weekday time.Weekday, hour int, location string) string {
	return fmt.Sprintf("%s on %s hour that starts at %d at %s\n", who, weekday, hour, location)
}

func (sp *SemesterPlan) String() string {
	s := fmt.Sprintf("From %d, month %d\n", sp.year, sp.fromMonth)
	s += fmt.Sprintf("Up to %d, month %d\n", sp.year, sp.uptoMonth)
	s += sp.ForAllWeekdays(infoline)
	return s
}

// Given an hour, gets information from all the person plans in the period plan
func (sp *SemesterPlan) GetHourEventStructs(t time.Time) HourEvents {

	hourevents := make(HourEvents, 0)

	// if not the right year
	if t.Year() != sp.year {
		return hourevents
	}

	// if not within the month range
	if !((t.Month() >= time.Month(sp.fromMonth)) && (t.Month() < time.Month(sp.uptoMonth))) {
		return hourevents
	}

	var hev HourEvent
	for _, persplan := range sp.personPlans {
		for _, wd := range persplan.workdays {

			// If not the right day of the week
			if wd.dayoftheweek != t.Weekday() {
				//fmt.Printf("Wrong day of the week! (%v and %v)\n", wd.dayoftheweek, t.Weekday())
				continue
			}

			// If not within the hour range
			if !((t.Hour() >= wd.fromHour) && (t.Hour() < wd.uptoHour)) {
				//fmt.Printf("Wrong hour range! (%v is not between %v and %v)\n", t.Hour(), wd.fromHour, wd.uptoHour)
				continue
			}

			// Found!
			hev.who = persplan.who
			hev.when = t
			hev.where = wd.location
			hourevents = append(hourevents, hev)
		}
	}

	// HourEvent structs
	return hourevents
}

func (sp *SemesterPlan) ViewHour(t time.Time) string {
	s := ""
	hourinfo := sp.GetHourEventStructs(t)
	for _, hev := range hourinfo {
		s += fmt.Sprintf("%s %s at %s, %v at hour %v\n", hev.when.String()[:10], hev.who, hev.where, hev.when.Weekday(), hev.when.Hour())
	}
	return s
}

func (sp *SemesterPlan) ViewDay(date time.Time) string {
	var t time.Time
	var hourString string
	s := ""
	for hour := FIRSTHOUR; hour <= LASTHOUR; hour++ {
		//fmt.Printf("hour: %d\n", hour)
		t = time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, time.UTC)
		hourString = sp.ViewHour(t)
		if hourString != "" {
			s += hourString + "\n"
		}
	}
	return s
}

// Make new plans, which is a collection of SemesterPlans
func NewPlans() *Plans {
	var plans Plans
	plans.all = make([]*SemesterPlan, 0)
	return &plans
}

// Add a SemesterPlan to the collection of plans
func (plans *Plans) AddSemesterPlan(sp *SemesterPlan) {
	plans.all = append(plans.all, sp)
}

// Get a string that is suitable to put in a table cell
func (plans *Plans) HTMLHourEvents(date time.Time) string {
	s := ""
	s2 := ""
	for _, sp := range plans.all {
		s2 = ""
		hourinfo := sp.GetHourEventStructs(date)
		for _, hev := range hourinfo {
			s2 += fmt.Sprintf("%s at %s%s", hev.who, hev.where, "<br>")
		}
		if s2 != "" {
			s += s2
		}
	}
	return s
}

// TODO: Create a function just like this that returns a list of HourEvent structs
func (plans *Plans) PrintHourEvents(date time.Time) {
	fmt.Printf("What's up at %s?\n", date.String())
	s := ""
	for _, pp := range plans.all {
		s += pp.ViewHour(date)
	}
	if s == "" {
		fmt.Println("Nothing!")
	} else {
		fmt.Println(s)
	}
}
