package genericsite

// OK, only admin stuff, 23-03-13

import (
	"strconv"
	"strings"

	. "github.com/xyproto/browserspeak"
	"github.com/xyproto/instapage"
	"github.com/xyproto/web"
)

// This part handles the "admin" pages

type AdminEngine struct {
	state *UserState
}

func NewAdminEngine(state *UserState) *AdminEngine {
	return &AdminEngine{state}
}

// Checks if the current user is logged in as Administrator right now
func (state *UserState) AdminRights(ctx *web.Context) bool {
	if username := GetBrowserUsername(ctx); username != "" {
		return state.IsLoggedIn(username) && state.IsAdministrator(username)
	}
	return false
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
func GenerateAdminStatus(state *UserState) SimpleContextHandle {
	return func(ctx *web.Context) string {
		if !state.AdminRights(ctx) {
			return "<div class=\"no\">Not logged in as Administrator</div>"
		}

		// TODO: List all sorts of info, edit users, etc
		s := "<h2>Administrator Dashboard</h2>"

		s += "<strong>User table</strong><br />"
		s += "<table class=\"whitebg\">"
		s += "<tr>"
		s += "<th>Username</th><th>Confirmed</th><th>Logged in</th><th>Administrator</th><th>Admin toggle</th><th>Remove user</th><th>Email</th><th>Password hash</th>"
		s += "</tr>"
		usernames, err := state.usernames.GetAll()
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
				s += TableCell(state.IsAdministrator(username))
				s += "<td><a class=\"darkgrey\" href=\"/admintoggle/" + username + "\">admin toggle</a></td>"
				// TODO: Ask for confirmation first with a instapage.MessageOKurl("blabla", "blabla", "/actually/remove/stuff")
				s += "<td><a class=\"careful\" href=\"/remove/" + username + "\">remove</a></td>"
				email, err := state.users.Get(username, "email")
				if err == nil {
					s += "<td>" + email + "</td>"
				}
				passwordHash, err := state.users.Get(username, "password")
				if err == nil {
					if strings.HasPrefix(passwordHash, "abc123") {
						s += "<td>" + passwordHash + " (<a href=\"/fixpassword/" + username + "\">fix</a>)</td>"
					} else {
						s += "<td>length " + strconv.Itoa(len(passwordHash)) + "</td>"
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
		usernames, err = state.unconfirmed.GetAll()
		if err == nil {
			for _, username := range usernames {
				s += "<tr>"
				s += "<td><a class=\"username\" href=\"/status/" + username + "\">" + username + "</a></td>"
				confirmationCode := state.GetConfirmationCode(username)
				s += "<td><a class=\"somewhatcareful\" href=\"/confirm/" + confirmationCode + "\">" + confirmationCode + "</a></td>"
				s += "<td><a class=\"careful\" href=\"/removeunconfirmed/" + username + "\">remove</a></td>"
				s += "</tr>"
			}
		}
		s += "</table>"
		return s
	}
}

// Checks if the given username is an administrator
func (state *UserState) IsAdministrator(username string) bool {
	if !state.HasUser(username) {
		return false
	}
	status, err := state.users.Get(username, "admin")
	if err != nil {
		return false
	}
	return TruthValue(status)
}

func GenerateStatusCurrentUser(state *UserState) SimpleContextHandle {
	return func(ctx *web.Context) string {
		if !state.AdminRights(ctx) {
			return instapage.MessageOKback("Status", "Not logged in as Administrator")
		}
		username := GetBrowserUsername(ctx)
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

func GenerateStatusUser(state *UserState) WebHandle {
	return func(ctx *web.Context, username string) string {
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
func GenerateRemoveUnconfirmedUser(state *UserState) WebHandle {
	return func(ctx *web.Context, username string) string {
		if !state.AdminRights(ctx) {
			return instapage.MessageOKback("Remove unconfirmed user", "Not logged in as Administrator")
		}

		if username == "" {
			return instapage.MessageOKback("Remove unconfirmed user", "Can't remove blank user.")
		}

		found := false
		usernames, err := state.unconfirmed.GetAll()
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

		// Remove the user
		state.unconfirmed.Del(username)

		// Remove additional data as well
		state.users.Del(username, "confirmationCode")

		return instapage.MessageOKurl("Remove unconfirmed user", "OK, removed "+username+" from the list of unconfirmed users.", "/admin")
	}
}

// TODO: Add possibility for Admin to restart the webserver

// TODO: Undo for removing users
// Remove a user
func GenerateRemoveUser(state *UserState) WebHandle {
	return func(ctx *web.Context, username string) string {
		if !state.AdminRights(ctx) {
			return instapage.MessageOKback("Remove user", "Not logged in as Administrator")
		}

		if username == "" {
			return instapage.MessageOKback("Remove user", "Can't remove blank user")
		}
		if !state.HasUser(username) {
			return instapage.MessageOKback("Remove user", username+" doesn't exists, could not remove")
		}

		// Remove the user
		state.usernames.Del(username)

		// Remove additional data as well
		state.users.Del(username, "loggedin")

		return instapage.MessageOKurl("Remove user", "OK, removed "+username, "/admin")
	}
}

func GenerateAllUsernames(state *UserState) SimpleContextHandle {
	return func(ctx *web.Context) string {
		if !state.AdminRights(ctx) {
			return instapage.MessageOKback("List usernames", "Not logged in as Administrator")
		}
		s := ""
		usernames, err := state.usernames.GetAll()
		if err == nil {
			for _, username := range usernames {
				s += username + "<br />"
			}
		}
		return instapage.MessageOKback("Usernames", s)
	}
}

//func GenerateGetCookie(state *UserState) SimpleContextHandle {
//	return func(ctx *web.Context) string {
//		if !state.AdminRights(ctx) {
//			return instapage.MessageOKback("Get cookie", "Not logged in as Administrator")
//		}
//		username := GetBrowserUsername(ctx)
//		return instapage.MessageOKback("Get cookie", "Cookie: username = "+username)
//	}
//}
//
//func GenerateSetCookie(state *UserState) WebHandle {
//	return func(ctx *web.Context, username string) string {
//		if !state.AdminRights(ctx) {
//			return instapage.MessageOKback("Set cookie", "Not logged in as Administrator")
//		}
//		if username == "" {
//			return instapage.MessageOKback("Set cookie", "Can't set cookie for empty username")
//		}
//		if !state.HasUser(username) {
//			return instapage.MessageOKback("Set cookie", "Can't store cookie for non-existsing user")
//		}
//		// Create a cookie that lasts for one hour,
//		// this is the equivivalent of a session for a given username
//		ctx.SetSecureCookiePath("user", username, 3600, "/")
//		return instapage.MessageOKback("Set cookie", "Cookie stored: user = "+username+".")
//	}
//}

func GenerateToggleAdmin(state *UserState) WebHandle {
	return func(ctx *web.Context, username string) string {
		if !state.AdminRights(ctx) {
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
		if !state.IsAdministrator(username) {
			state.users.Set(username, "admin", "true")
			return instapage.MessageOKurl("Admin toggle", "OK, "+username+" is now an admin", "/admin")
		}
		state.users.Set(username, "admin", "false")
		return instapage.MessageOKurl("Admin toggle", "OK, "+username+" is now a regular user", "/admin")
	}
}

// This is now deprecated. Keep it around only as a nice example of fixing user values that worked.
//func GenerateFixPassword(state *UserState) WebHandle {
//	return func(ctx *web.Context, username string) string {
//		if !state.AdminRights(ctx) {
//			return instapage.MessageOKback("Fix password", "Not logged in as Administrator")
//		}
//		if username == "" {
//			return instapage.MessageOKback("Fix password", "Can't fix empty username")
//		}
//		if !state.HasUser(username) {
//			return instapage.MessageOKback("Fix password", "Can't fix non-existing user")
//		}
//		password := ""
//		passwordHash, err := state.users.Get(username, "password")
//		if err != nil {
//			return instapage.MessageOKback("Fix password", "Could not retrieve password hash")
//		}
//		if strings.HasPrefix(passwordHash, "abc123") {
//			if strings.HasSuffix(passwordHash, "abc123") {
//				password = passwordHash[6 : len(passwordHash)-6]
//			}
//		}
//		newPasswordHash := HashPasswordVersion2(password)
//		state.users.Set(username, "password", newPasswordHash)
//		return instapage.MessageOKurl("Fix password", "Ok, upgraded the password hash for "+username+" to version 2.", "/admin")
//	}
//}

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
	//web.Get("/cookie/get", GenerateGetCookie(state))
	//web.Get("/cookie/set/(.*)", GenerateSetCookie(state))
	//web.Get("/fixpassword/(.*)", GenerateFixPassword(state))
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

`
		//
	}
}
