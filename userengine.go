package siteengines

// TODO: Check for the visual likeness of two usernames when checking for availability! Generate images and compare pixels.
// TODO: Logging in should work case sensitively, or at least without concern for the case of the first letter
// TODO: Consider using "0" and "1" instead of "true" or "false" when setting values, while still understanding "true" or "false"
// TODO: The password should be set at confirmation time instead of registration-time in order to make the process clearer and pave the way for invite-only?

import (
	"net/http"
	"strings"

	. "github.com/xyproto/genericsite"
	"github.com/xyproto/instapage"
	. "github.com/xyproto/onthefly"
	"github.com/xyproto/permissions"
	. "github.com/xyproto/webhandle"
)

type UserEngine struct {
	state *permissions.UserState
}

func NewUserEngine(state *permissions.UserState) *UserEngine {
	return &UserEngine{state}
}

// Create a user by adding the username to the list of usernames
func (ue *UserEngine) GenerateConfirmUser() http.HandlerFunc {
	state := ue.state
	return func(w http.ResponseWriter, req *http.Request) {
		val := GetLast(req.URL)

		confirmationCode := val

		unconfirmedUsernames, err := state.GetAllUnconfirmedUsernames()
		if err != nil {
			Ret(w, instapage.MessageOKurl("Confirmation", "All users are confirmed already.", "/register"))
			return
		}

		// Find the username by looking up the confirmationCode on unconfirmed users
		username := ""
		for _, aUsername := range unconfirmedUsernames {
			aConfirmationCode, err := state.GetConfirmationCode(aUsername)
			if err != nil {
				// If the confirmation code can not be found, just skip this one
				continue
			}
			if confirmationCode == aConfirmationCode {
				// Found the right user
				username = aUsername
				break
			}
		}

		// Check that the user is there
		if username == "" {
			// Say "no longer" because we don't care about people that just try random confirmation links
			Ret(w, instapage.MessageOKurl("Confirmation", "The confirmation link is no longer valid.", "/register"))
			return
		}
		hasUser := state.HasUser(username)
		if !hasUser {
			Ret(w, instapage.MessageOKurl("Confirmation", "The user you wish to confirm does not exist anymore.", "/register"))
			return
		}

		// Remove from the list of unconfirmed usernames
		state.RemoveUnconfirmed(username)

		// Mark user as confirmed
		state.MarkConfirmed(username)

		Ret(w, instapage.MessageOKurl("Confirmation", "Thank you "+username+", you can now log in.", "/login"))
	}
}

// Log in a user by changing the loggedin value
func (ue *UserEngine) GenerateLoginUser() http.HandlerFunc {
	state := ue.state
	return func(w http.ResponseWriter, req *http.Request) {
		// Get passwrod from url (should be from POST fields instead?)
		password := GetFormParam(req, "password")

		if password == "" {
			Ret(w, instapage.MessageOKback("Login", "Can't log in without a password."))
			return
		}
		username := GetLast(req.URL)
		if username == "" {
			Ret(w, instapage.MessageOKback("Login", "Can't log in with a blank username."))
			return
		}
		if !state.HasUser(username) {
			Ret(w, instapage.MessageOKback("Login", "User "+username+" does not exist, could not log in."))
			return
		}
		if !state.IsConfirmed(username) {
			Ret(w, instapage.MessageOKback("Login", "The email for "+username+" has not been confirmed, check your email and follow the link."))
			return
		}
		if !state.CorrectPassword(username, password) {
			Ret(w, instapage.MessageOKback("Login", "Wrong password."))
			return
		}

		// Log in the user by changing the database and setting a secure cookie
		state.SetLoggedIn(username)

		// Also store the username in the browser
		state.SetUsernameCookie(w, username)

		// TODO: Use a welcoming messageOK where the user can see when he/she last logged in and from which host

		if username == "admin" {
			w.Header().Set("Refresh", "0; url=/admin")
		} else {
			// TODO: Redirect to the page the user was at before logging in
			w.Header().Set("Refresh", "0; url=/")
		}

		return
	}
}

// TODO: Forgot username? Enter email, send username.
// TODO: Lost confirmation link? Enter mail, Receive confirmation link.
// TODO: Forgot password? Enter mail, receive reset-password link.
// TODO: Make sure not two usernames can register at once before confirming
// TODO: Only one username per email address? (meh? can use more than one address?=
// TODO: Maximum 1 confirmation email per email adress
// TODO: Maximum 1 forgot username per email adress per day
// TODO: Maximum 1 forgot password per email adress per day
// TODO: Maximum 1 lost confirmation link per email adress per day
// TODO: Link for "Did you not request this email? Click here" i alle eposter som sendes.
// TODO: Rate limiting, maximum rate per minute or day

// Register a new user, site is ie. "archlinux.no"
func (ue *UserEngine) GenerateRegisterUser(site string) http.HandlerFunc {
	state := ue.state
	return func(w http.ResponseWriter, req *http.Request) {

		// Password checks
		password1 := GetFormParam(req, "password1")
		if password1 == "" {
			Ret(w, instapage.MessageOKback("Register", "Can't register without a password."))
			return
		}
		password2 := GetFormParam(req, "password2")
		if password2 == "" {
			Ret(w, instapage.MessageOKback("Register", "Please confirm the password by typing it in twice."))
			return
		}
		if password1 != password2 {
			Ret(w, instapage.MessageOKback("Register", "The password and confirmation password must be equal."))
			return
		}

		// Email checks
		email := GetFormParam(req, "email")
		if email == "" {
			Ret(w, instapage.MessageOKback("Register", "Can't register without an email address."))
			return
		}
		// must have @ and ., but no " "
		if !strings.Contains(email, "@") || !strings.Contains(email, ".") || strings.Contains(email, " ") {
			Ret(w, instapage.MessageOKback("Register", "Please use a valid email address."))
			return
		}
		if email != CleanUserInput(email) {
			Ret(w, instapage.MessageOKback("Register", "The sanitized email differs from the given email."))
			return
		}

		// Username checks
		username := GetLast(req.URL)
		if username == "" {
			Ret(w, instapage.MessageOKback("Register", "Can't register without a username."))
			return
		}
		if state.HasUser(username) {
			Ret(w, instapage.MessageOKback("Register", "That user already exists, try another username."))
			return
		}

		// Only some letters are allowed in the username
		err := Check(username, password1)
		if err != nil {
			Ret(w, instapage.MessageOKback("Register", err.Error()))
			return
		}

		adminuser := false
		// A special case
		if username == "admin" {
			// The first user to register with the username "admin" becomes the administrator
			adminuser = true
		}

		// Register the user
		state.AddUser(username, password1, email)

		// Mark user as administrator if that is the case
		if adminuser {
			// Set admin status
			state.SetAdminStatus(username)
		}

		confirmationCode, err := state.GenerateUniqueConfirmationCode()
		if err != nil {
			panic(err.Error())
		}

		// If registering the admin user (first user on the system), don't send a confirmation email, just register it
		if adminuser {

			// Mark user as confirmed
			state.MarkConfirmed(username)

			// Redirect
			Ret(w, instapage.MessageOKurl("Registration complete", "Thanks for registering, the admin user has been created.", "/login"))
			return

		}

		// Send confirmation email
		ConfirmationEmail(site, "https://"+site+"/confirm/"+confirmationCode, username, email)

		// Register the need to be confirmed
		state.AddUnconfirmed(username, confirmationCode)

		// Redirect
		Ret(w, instapage.MessageOKurl("Registration complete", "Thanks for registering, the confirmation e-mail has been sent.", "/login"))
	}
}

// Log out a user by changing the loggedin value
func (ue *UserEngine) GenerateLogoutCurrentUser() http.HandlerFunc {
	state := ue.state
	return func(w http.ResponseWriter, req *http.Request) {
		username := state.GetUsername(req)
		if username == "" {
			Ret(w, instapage.MessageOKback("Logout", "No user to log out"))
			return
		}
		if !state.HasUser(username) {
			Ret(w, instapage.MessageOKback("Logout", "user "+username+" does not exist, could not log out"))
			return
		}

		// Log out the user by changing the database, the cookie can stay
		state.SetLoggedOut(username)

		// Redirect
		//ctx.SetHeader("Refresh", "0; url=/login", true)
		Ret(w, instapage.MessageOKurl("Logout", username+" is now logged out. Hope to see you soon!", "/login"))
	}
}

func GenerateNoJavascriptMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		Ret(w, instapage.MessageOKback("JavaScript error", "Cookies and Javascript must be enabled.<br />Older browsers might be supported in the future."))
		return
	}
}

func LoginCP(basecp BaseCP, state *permissions.UserState, url string) *ContentPage {
	cp := basecp(state)
	cp.ContentTitle = "Login"
	cp.ContentHTML = instapage.LoginForm()
	cp.ContentJS += OnClick("#loginButton", "$('#loginForm').get(0).setAttribute('action', '/login/' + $('#username').val());")
	//cp.ExtraCSSurls = append(cp.ExtraCSSurls, "/css/login.css")
	cp.Url = url

	// Hide the Login menu if we're on the Login page
	//cp.HeaderJS = strings.Replace(cp.HeaderJS, "menuLogin", "menuNop", 1)
	//cp.ContentJS += Hide("#menuLogin")

	return cp
}

func RegisterCP(basecp BaseCP, state *permissions.UserState, url string) *ContentPage {
	cp := basecp(state)
	cp.ContentTitle = "Register"
	cp.ContentHTML = instapage.RegisterForm()
	cp.ContentJS += OnClick("#registerButton", "$('#registerForm').get(0).setAttribute('action', '/register/' + $('#username').val());")
	//cp.ExtraCSSurls = append(cp.ExtraCSSurls, "/css/register.css")
	cp.Url = url

	// Hide the Register menu if we're on the Register page
	//cp.HeaderJS = strings.Replace(cp.HeaderJS, "menuRegister", "menuNop", 1)
	//cp.ContentJS += Hide("#menuRegister")

	return cp
}

// Site is ie. "archlinux.no" and used for sending confirmation emails
func (ue *UserEngine) ServePages(mux *http.ServeMux, site string) {
	mux.HandleFunc("/register/", ue.GenerateRegisterUser(site))
	mux.HandleFunc("/register", GenerateNoJavascriptMessage())
	mux.HandleFunc("/login/", ue.GenerateLoginUser())
	mux.HandleFunc("/login", GenerateNoJavascriptMessage())
	mux.HandleFunc("/logout", ue.GenerateLogoutCurrentUser())
	mux.HandleFunc("/confirm/", ue.GenerateConfirmUser())
}
