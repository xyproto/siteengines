package siteengines

import (
	"fmt"
	"net/http"
	"strings"

	. "github.com/xyproto/genericsite"
	"github.com/xyproto/instapage"
	"github.com/xyproto/permissions"
	"github.com/xyproto/symbolhash"
)

// This part handles the "admin" pages

type AdminEngine struct {
	state *permissions.UserState
}

func NewAdminEngine(state *permissions.UserState) *AdminEngine {
	return &AdminEngine{state}
}

func (ae *AdminEngine) ServePages(mux *http.ServeMux, basecp BaseCP, menuEntries MenuEntries) {
	ae.serveSystem()

	state := ae.state

	adminCP := basecp(state)
	adminCP.ContentTitle = "Admin"
	adminCP.ExtraCSSurls = append(adminCP.ExtraCSSurls, "/css/admin.css")

	// template content generator
	tpvf := DynamicMenuFactoryGenerator(menuEntries)

	mux.HandleFunc("/admin", adminCP.GetHandle(GenerateAdminStatus(state), tpvf(state)))
	mux.HandleFunc("/css/admin.css", ae.GenerateCSS(adminCP.ColorScheme))
}

// TODO: Log and graph when people visit pages and when people contribute content
// This one is wrapped by ServeAdminPages
func GenerateAdminStatus(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) string {
		if !state.AdminRights(req) {
			return "<div class=\"no\">Not logged in as Administrator</div>"
		}

		// TODO: List all sorts of info, edit users, etc
		s := "<h2>Administrator Dashboard</h2>"

		s += "<strong>User table</strong><br />"
		s += "<table class=\"whitebg\">"
		s += "<tr>"
		s += "<th>Username</th><th>Confirmed</th><th>Logged in</th><th>Administrator</th><th>Admin toggle</th><th>Remove user</th><th>Email</th><th>Password hash</th>"
		s += "</tr>"
		usernames, err := state.GetAllUsernames()
		if err == nil {
			for rownr, username := range usernames {
				if rownr%2 == 0 {
					s += "<tr class=\"even\">"
				} else {
					s += "<tr class=\"odd\">"
				}
				s += "<td><a class=\"username\" href=\"/status/" + username + "\">" + username + "</a></td>"
				s += TableCell(state.IsConfirmed(username))
				s += TableCell(state.IsLoggedIn(username))
				s += TableCell(state.IsAdmin(username))
				s += "<td><a class=\"darkgrey\" href=\"/admintoggle/" + username + "\">admin toggle</a></td>"
				// TODO: Ask for confirmation first with a instapage.MessageOKurl("blabla", "blabla", "/actually/remove/stuff")
				s += "<td><a class=\"careful\" href=\"/remove/" + username + "\">remove</a></td>"
				email, err := state.GetEmail(username)
				if err == nil {
					// The cleanup happens at registration time, but it's ok with an extra cleanup
					s += "<td>" + CleanUserInput(email) + "</td>"
				}
				passwordHash, err := state.GetPasswordHash(username)
				if err == nil {
					if strings.HasPrefix(passwordHash, "abc123") {
						s += "<td>" + passwordHash + " (<a href=\"/fixpassword/" + username + "\">fix</a>)</td>"
					} else {
						s += "<td>" + symbolhash.New(passwordHash, 16).String() + "</td>"
					}
				}
				s += "</tr>"
			}
		}
		s += "</table>"
		s += "<br />"
		s += "<strong>Unconfirmed users</strong><br />"
		s += "<table>"
		s += "<tr>"
		s += "<th>Username</th><th>Confirmation link</th><th>Remove</th>"
		s += "</tr>"
		usernames, err = state.GetAllUnconfirmedUsernames()
		if err == nil {
			for _, username := range usernames {
				s += "<tr>"
				s += "<td><a class=\"username\" href=\"/status/" + username + "\">" + username + "</a></td>"
				confirmationCode, err := state.GetConfirmationCode(username)
				if err != nil {
					panic("ERROR: Could not get confirmation code")
				}
				s += "<td><a class=\"somewhatcareful\" href=\"/confirm/" + confirmationCode + "\">" + confirmationCode + "</a></td>"
				s += "<td><a class=\"careful\" href=\"/removeunconfirmed/" + username + "\">remove</a></td>"
				s += "</tr>"
			}
		}
		s += "</table>"
		return s
	}
}

func GenerateStatusCurrentUser(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) string {
		if !state.AdminRights(req) {
			return instapage.MessageOKback("Status", "Not logged in as Administrator")
		}
		username := state.GetUsername(req)
		if username == "" {
			return instapage.MessageOKback("Current user status", "No user logged in")
		}
		hasUser := state.HasUser(username)
		if !hasUser {
			return instapage.MessageOKback("Current user status", username+" does not exist")
		}
		if !(state.IsLoggedIn(username)) {
			return instapage.MessageOKback("Current user status", "User "+username+" is not logged in")
		}
		return instapage.MessageOKback("Current user status", "User "+username+" is logged in")
	}
}

func GenerateStatusUser(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request, username string) string {
		if username == "" {
			return instapage.MessageOKback("Status", "No username given")
		}
		if !state.HasUser(username) {
			return instapage.MessageOKback("Status", username+" does not exist")
		}
		loggedinStatus := "not logged in"
		if state.IsLoggedIn(username) {
			loggedinStatus = "logged in"
		}
		confirmStatus := "email has not been confirmed"
		if state.IsConfirmed(username) {
			confirmStatus = "email has been confirmed"
		}
		return instapage.MessageOKback("Status", username+" is "+loggedinStatus+" and "+confirmStatus)
	}
}

// Remove an unconfirmed user
func GenerateRemoveUnconfirmedUser(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request, username string) string {
		if !state.AdminRights(req) {
			return instapage.MessageOKback("Remove unconfirmed user", "Not logged in as Administrator")
		}

		if username == "" {
			return instapage.MessageOKback("Remove unconfirmed user", "Can't remove blank user.")
		}

		found := false
		usernames, err := state.GetAllUnconfirmedUsernames()
		if err == nil {
			for _, unconfirmedUsername := range usernames {
				if username == unconfirmedUsername {
					found = true
					break
				}
			}
		}

		if !found {
			return instapage.MessageOKback("Remove unconfirmed user", "Can't find "+username+" in the list of unconfirmed users.")
		}

		// Mark as confirmed
		state.RemoveUnconfirmed(username)

		return instapage.MessageOKurl("Remove unconfirmed user", "OK, removed "+username+" from the list of unconfirmed users.", "/admin")
	}
}

// TODO: Add possibility for Admin to restart the webserver

// TODO: Undo for removing users
// Remove a user
func GenerateRemoveUser(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request, username string) string {
		if !state.AdminRights(req) {
			return instapage.MessageOKback("Remove user", "Not logged in as Administrator")
		}

		if username == "" {
			return instapage.MessageOKback("Remove user", "Can't remove blank user")
		}
		if !state.HasUser(username) {
			return instapage.MessageOKback("Remove user", username+" doesn't exists, could not remove")
		}

		// Remove the user
		state.RemoveUser(username)

		return instapage.MessageOKurl("Remove user", "OK, removed "+username, "/admin")
	}
}

func GenerateAllUsernames(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) string {
		if !state.AdminRights(req) {
			return instapage.MessageOKback("List usernames", "Not logged in as Administrator")
		}
		s := ""
		usernames, err := state.GetAllUsernames()
		if err == nil {
			for _, username := range usernames {
				s += username + "<br />"
			}
		}
		return instapage.MessageOKback("Usernames", s)
	}
}

func GenerateToggleAdmin(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request, username string) string {
		if !state.AdminRights(req) {
			return instapage.MessageOKback("Admin toggle", "Not logged in as Administrator")
		}
		if username == "" {
			return instapage.MessageOKback("Admin toggle", "Can't set toggle empty username")
		}
		if !state.HasUser(username) {
			return instapage.MessageOKback("Admin toggle", "Can't toggle non-existing user")
		}
		// A special case
		if username == "admin" {
			return instapage.MessageOKback("Admin toggle", "Can't remove admin rights from the admin user")
		}
		if !state.IsAdmin(username) {
			state.SetAdminStatus(username)
			return instapage.MessageOKurl("Admin toggle", "OK, "+username+" is now an admin", "/admin")
		}
		state.RemoveAdminStatus(username)
		return instapage.MessageOKurl("Admin toggle", "OK, "+username+" is now a regular user", "/admin")
	}
}

func (ae *AdminEngine) serveSystem() {
	state := ae.state

	// These are available for everyone
	mux.HandleFunc("/status/(.*)", GenerateStatusUser(state))

	// These are only available as administrator, all have checks
	mux.HandleFunc("/status", GenerateStatusCurrentUser(state))
	mux.HandleFunc("/remove/(.*)", GenerateRemoveUser(state))
	mux.HandleFunc("/removeunconfirmed/(.*)", GenerateRemoveUnconfirmedUser(state))
	mux.HandleFunc("/users/(.*)", GenerateAllUsernames(state))
	mux.HandleFunc("/admintoggle/(.*)", GenerateToggleAdmin(state))
}

func (ae *AdminEngine) GenerateCSS(cs *ColorScheme) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/css")
		// TODO: Consider if menus should be hidden this way when visiting a subpage
		//#menuAdmin {
		//	display: none;
		//}
		fmt.Fprint(w, `
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
