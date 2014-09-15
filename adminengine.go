package siteengines

import (
	"net/http"
	"strings"

	. "github.com/xyproto/genericsite"
	"github.com/xyproto/instapage"
	"github.com/xyproto/permissions"
	"github.com/xyproto/symbolhash"
	. "github.com/xyproto/webhandle"
)

// This part handles the "admin" pages

type AdminEngine struct {
	state *permissions.UserState
}

func NewAdminEngine(state *permissions.UserState) *AdminEngine {
	return &AdminEngine{state}
}

func (ae *AdminEngine) ServePages(mux *http.ServeMux, basecp BaseCP, menuEntries MenuEntries) {
	ae.serveSystem(mux)

	state := ae.state

	adminCP := basecp(state)
	adminCP.ContentTitle = "Admin"
	adminCP.ExtraCSSurls = append(adminCP.ExtraCSSurls, "/css/admin.css")

	// template content generator
	tpvf := DynamicMenuFactoryGenerator(menuEntries)

	mux.HandleFunc("/admin", adminCP.WrapHandle(mux, GenerateAdminStatus(state), tpvf(state)))
	mux.HandleFunc("/css/admin.css", ae.GenerateCSS(adminCP.ColorScheme))
}

// TODO: Log and graph when people visit pages and when people contribute content
// This one is wrapped by ServeAdminPages
func GenerateAdminStatus(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if !state.AdminRights(req) {
			Ret(w, "<div class=\"no\">Not logged in as Administrator</div>")
			return
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
		Ret(w, s)
	}
}

func GenerateStatusCurrentUser(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if !state.AdminRights(req) {
			Ret(w, instapage.MessageOKback("Status", "Not logged in as Administrator"))
			return
		}
		username := state.GetUsername(req)
		if username == "" {
			Ret(w, instapage.MessageOKback("Current user status", "No user logged in"))
			return
		}
		hasUser := state.HasUser(username)
		if !hasUser {
			Ret(w, instapage.MessageOKback("Current user status", username+" does not exist"))
			return
		}
		if !(state.IsLoggedIn(username)) {
			Ret(w, instapage.MessageOKback("Current user status", "User "+username+" is not logged in"))
			return
		}
		Ret(w, instapage.MessageOKback("Current user status", "User "+username+" is logged in"))
	}
}

func GenerateStatusUser(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Fetch the username from the last part of the URL path
		username := GetLast(req.URL)

		if username == "" {
			Ret(w, instapage.MessageOKback("Status", "No username given"))
			return
		}
		if !state.HasUser(username) {
			Ret(w, instapage.MessageOKback("Status", username+" does not exist"))
			return
		}
		loggedinStatus := "not logged in"
		if state.IsLoggedIn(username) {
			loggedinStatus = "logged in"
		}
		confirmStatus := "email has not been confirmed"
		if state.IsConfirmed(username) {
			confirmStatus = "email has been confirmed"
		}
		Ret(w, instapage.MessageOKback("Status", username+" is "+loggedinStatus+" and "+confirmStatus))
		return
	}
}

// Remove an unconfirmed user
func GenerateRemoveUnconfirmedUser(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Fetch the username from the last part of the URL path
		username := GetLast(req.URL)

		if !state.AdminRights(req) {
			Ret(w, instapage.MessageOKback("Remove unconfirmed user", "Not logged in as Administrator"))
			return
		}

		if username == "" {
			Ret(w, instapage.MessageOKback("Remove unconfirmed user", "Can't remove blank user."))
			return
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
			Ret(w, instapage.MessageOKback("Remove unconfirmed user", "Can't find "+username+" in the list of unconfirmed users."))
			return
		}

		// Mark as confirmed
		state.RemoveUnconfirmed(username)

		Ret(w, instapage.MessageOKurl("Remove unconfirmed user", "OK, removed "+username+" from the list of unconfirmed users.", "/admin"))
		return
	}
}

// TODO: Add possibility for Admin to restart the webserver

// TODO: Undo for removing users
// Remove a user
func GenerateRemoveUser(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Fetch the username from the last part of the URL path
		username := GetLast(req.URL)

		if !state.AdminRights(req) {
			Ret(w, instapage.MessageOKback("Remove user", "Not logged in as Administrator"))
			return
		}

		if username == "" {
			Ret(w, instapage.MessageOKback("Remove user", "Can't remove blank user"))
			return
		}
		if !state.HasUser(username) {
			Ret(w, instapage.MessageOKback("Remove user", username+" doesn't exists, could not remove"))
			return
		}

		// Remove the user
		state.RemoveUser(username)

		Ret(w, instapage.MessageOKurl("Remove user", "OK, removed "+username, "/admin"))
	}
}

func GenerateAllUsernames(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if !state.AdminRights(req) {
			Ret(w, instapage.MessageOKback("List usernames", "Not logged in as Administrator"))
			return
		}
		s := ""
		usernames, err := state.GetAllUsernames()
		if err == nil {
			for _, username := range usernames {
				s += username + "<br />"
			}
		}
		Ret(w, instapage.MessageOKback("Usernames", s))
	}
}

func GenerateToggleAdmin(state *permissions.UserState) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		// Fetch the username from the last part of the URL path
		username := GetLast(req.URL)

		if !state.AdminRights(req) {
			Ret(w, instapage.MessageOKback("Admin toggle", "Not logged in as Administrator"))
			return
		}
		if username == "" {
			Ret(w, instapage.MessageOKback("Admin toggle", "Can't set toggle empty username"))
			return
		}
		if !state.HasUser(username) {
			Ret(w, instapage.MessageOKback("Admin toggle", "Can't toggle non-existing user"))
			return
		}
		// A special case
		if username == "admin" {
			Ret(w, instapage.MessageOKback("Admin toggle", "Can't remove admin rights from the admin user"))
			return
		}
		if !state.IsAdmin(username) {
			state.SetAdminStatus(username)
			Ret(w, instapage.MessageOKurl("Admin toggle", "OK, "+username+" is now an admin", "/admin"))
			return
		}
		state.RemoveAdminStatus(username)
		Ret(w, instapage.MessageOKurl("Admin toggle", "OK, "+username+" is now a regular user", "/admin"))
	}
}

func (ae *AdminEngine) serveSystem(mux *http.ServeMux) {
	state := ae.state

	// These are available for everyone
	mux.HandleFunc("/status/", GenerateStatusUser(state))

	// These are only available as administrator, all have checks
	mux.HandleFunc("/status", GenerateStatusCurrentUser(state))
	mux.HandleFunc("/remove/", GenerateRemoveUser(state))
	mux.HandleFunc("/removeunconfirmed/", GenerateRemoveUnconfirmedUser(state))
	mux.HandleFunc("/users/", GenerateAllUsernames(state))
	mux.HandleFunc("/admintoggle/", GenerateToggleAdmin(state))
}

func (ae *AdminEngine) GenerateCSS(cs *ColorScheme) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/css")
		// TODO: Consider if menus should be hidden this way when visiting a subpage
		//#menuAdmin {
		//	display: none;
		//}
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
