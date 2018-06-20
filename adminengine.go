package siteengines

import (
	"strings"

	"github.com/hoisie/web"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/symbolhash"
	. "github.com/xyproto/webhandle"
)

// This part handles the "admin" pages

type AdminEngine struct {
	state pinterface.IUserState
}

func NewAdminEngine(state pinterface.IUserState) (*AdminEngine, error) {
	return &AdminEngine{state}, nil
}

func (ae *AdminEngine) ServePages(basecp BaseCP, menuEntries MenuEntries) {
	ae.serveSystem()

	state := ae.state

	adminCP := basecp(state)
	adminCP.ContentTitle = "Admin"
	adminCP.ExtraCSSurls = append(adminCP.ExtraCSSurls, "/css/admin.css")

	// template content generator
	tpvf := DynamicMenuFactoryGenerator(menuEntries)

	web.Get("/admin", adminCP.WrapSimpleContextHandle(GenerateAdminStatus(state), tpvf(state)))
	web.Get("/css/admin.css", ae.GenerateCSS(adminCP.ColorScheme))
}

// TODO: Log and graph when people visit pages and when people contribute content
// This one is wrapped by ServeAdminPages
func GenerateAdminStatus(state pinterface.IUserState) SimpleContextHandle {
	return func(ctx *web.Context) string {
		if !state.AdminRights(ctx.Request) {
			return "<div class=\"no\">Not logged in as Administrator</div>"
		}

		// TODO: List all sorts of info, edit users, etc
		s := "<h2>Administrator Dashboard</h2>"

		s += "<strong>User table</strong><br />"
		s += "<table class=\"whitebg\">"
		s += "<tr>"
		s += "<th>Username</th><th>Confirmed</th><th>Logged in</th><th>Administrator</th><th>Admin toggle</th><th>Remove user</th><th>Email</th><th>Password hash</th>"
		s += "</tr>"
		usernames, err := state.AllUsernames()
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
				// TODO: Ask for confirmation first with a MessageOKurl("blabla", "blabla", "/actually/remove/stuff")
				s += "<td><a class=\"careful\" href=\"/remove/" + username + "\">remove</a></td>"
				email, err := state.Email(username)
				if err == nil {
					// The cleanup happens at registration time, but it's ok with an extra cleanup
					s += "<td>" + CleanUserInput(email) + "</td>"
				}
				passwordHash, err := state.PasswordHash(username)
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
		usernames, err = state.AllUnconfirmedUsernames()
		if err == nil {
			for _, username := range usernames {
				s += "<tr>"
				s += "<td><a class=\"username\" href=\"/status/" + username + "\">" + username + "</a></td>"
				confirmationCode, err := state.ConfirmationCode(username)
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

func GenerateStatusCurrentUser(state pinterface.IUserState) SimpleContextHandle {
	return func(ctx *web.Context) string {
		if !state.AdminRights(ctx.Request) {
			return MessageOKback("Status", "Not logged in as Administrator")
		}
		username := state.Username(ctx.Request)
		if username == "" {
			return MessageOKback("Current user status", "No user logged in")
		}
		hasUser := state.HasUser(username)
		if !hasUser {
			return MessageOKback("Current user status", username+" does not exist")
		}
		if !(state.IsLoggedIn(username)) {
			return MessageOKback("Current user status", "User "+username+" is not logged in")
		}
		return MessageOKback("Current user status", "User "+username+" is logged in")
	}
}

func GenerateStatusUser(state pinterface.IUserState) WebHandle {
	return func(ctx *web.Context, username string) string {
		if username == "" {
			return MessageOKback("Status", "No username given")
		}
		if !state.HasUser(username) {
			return MessageOKback("Status", username+" does not exist")
		}
		loggedinStatus := "not logged in"
		if state.IsLoggedIn(username) {
			loggedinStatus = "logged in"
		}
		confirmStatus := "email has not been confirmed"
		if state.IsConfirmed(username) {
			confirmStatus = "email has been confirmed"
		}
		return MessageOKback("Status", username+" is "+loggedinStatus+" and "+confirmStatus)
	}
}

// Remove an unconfirmed user
func GenerateRemoveUnconfirmedUser(state pinterface.IUserState) WebHandle {
	return func(ctx *web.Context, username string) string {
		if !state.AdminRights(ctx.Request) {
			return MessageOKback("Remove unconfirmed user", "Not logged in as Administrator")
		}

		if username == "" {
			return MessageOKback("Remove unconfirmed user", "Can't remove blank user.")
		}

		found := false
		usernames, err := state.AllUnconfirmedUsernames()
		if err == nil {
			for _, unconfirmedUsername := range usernames {
				if username == unconfirmedUsername {
					found = true
					break
				}
			}
		}

		if !found {
			return MessageOKback("Remove unconfirmed user", "Can't find "+username+" in the list of unconfirmed users.")
		}

		// Mark as confirmed
		state.RemoveUnconfirmed(username)

		return MessageOKurl("Remove unconfirmed user", "OK, removed "+username+" from the list of unconfirmed users.", "/admin")
	}
}

// TODO: Add possibility for Admin to restart the webserver

// TODO: Undo for removing users
// Remove a user
func GenerateRemoveUser(state pinterface.IUserState) WebHandle {
	return func(ctx *web.Context, username string) string {
		if !state.AdminRights(ctx.Request) {
			return MessageOKback("Remove user", "Not logged in as Administrator")
		}

		if username == "" {
			return MessageOKback("Remove user", "Can't remove blank user")
		}
		if !state.HasUser(username) {
			return MessageOKback("Remove user", username+" doesn't exists, could not remove")
		}

		// Remove the user
		state.RemoveUser(username)

		return MessageOKurl("Remove user", "OK, removed "+username, "/admin")
	}
}

func GenerateAllUsernames(state pinterface.IUserState) SimpleContextHandle {
	return func(ctx *web.Context) string {
		if !state.AdminRights(ctx.Request) {
			return MessageOKback("List usernames", "Not logged in as Administrator")
		}
		s := ""
		usernames, err := state.AllUsernames()
		if err == nil {
			for _, username := range usernames {
				s += username + "<br />"
			}
		}
		return MessageOKback("Usernames", s)
	}
}

func GenerateToggleAdmin(state pinterface.IUserState) WebHandle {
	return func(ctx *web.Context, username string) string {
		if !state.AdminRights(ctx.Request) {
			return MessageOKback("Admin toggle", "Not logged in as Administrator")
		}
		if username == "" {
			return MessageOKback("Admin toggle", "Can't set toggle empty username")
		}
		if !state.HasUser(username) {
			return MessageOKback("Admin toggle", "Can't toggle non-existing user")
		}
		// A special case
		if username == "admin" {
			return MessageOKback("Admin toggle", "Can't remove admin rights from the admin user")
		}
		if !state.IsAdmin(username) {
			state.SetAdminStatus(username)
			return MessageOKurl("Admin toggle", "OK, "+username+" is now an admin", "/admin")
		}
		state.RemoveAdminStatus(username)
		return MessageOKurl("Admin toggle", "OK, "+username+" is now a regular user", "/admin")
	}
}

func (ae *AdminEngine) serveSystem() {
	state := ae.state

	// These are available for everyone
	web.Get("/status/(.*)", GenerateStatusUser(state))

	// These are only available as administrator, all have checks
	web.Get("/status", GenerateStatusCurrentUser(state))
	web.Get("/remove/(.*)", GenerateRemoveUser(state))
	web.Get("/removeunconfirmed/(.*)", GenerateRemoveUnconfirmedUser(state))
	web.Get("/users/(.*)", GenerateAllUsernames(state))
	web.Get("/admintoggle/(.*)", GenerateToggleAdmin(state))
}

func (ae *AdminEngine) GenerateCSS(cs *ColorScheme) SimpleContextHandle {
	return func(ctx *web.Context) string {
		ctx.ContentType("css")
		// TODO: Consider if menus should be hidden this way when visiting a subpage
		//#menuAdmin {
		//	display: none;
		//}
		return `
.even {
	background-color: #a0a0a0;
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

`
		//
	}
}
