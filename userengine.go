package siteengines

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/hoisie/web"
	"github.com/xyproto/cookie"
	. "github.com/xyproto/genericsite"
	. "github.com/xyproto/onthefly"
	"github.com/xyproto/pinterface"
	. "github.com/xyproto/webhandle"
)

var (
	charErr  = errors.New("Only letters, numbers and underscore are allowed in usernames.")
	equalErr = errors.New("Username and password must be different, try another password.")
)

// Check that the given username and password are different.
// Also check if the chosen username only contains letters, numbers and/or underscore.
// Use the "CorrectPassword" function for checking if the password is correct.
func ValidUsernamePassword(username, password string) error {
	const allAllowedLetters = "abcdefghijklmnopqrstuvwxyzæøåABCDEFGHIJKLMNOPQRSTUVWXYZÆØÅ_0123456789"
NEXT:
	for _, letter := range username {
		for _, allowedLetter := range allAllowedLetters {
			if letter == allowedLetter {
				continue NEXT // check the next letter in the username
			}
		}
		return charErr
	}
	if username == password {
		return equalErr
	}
	return nil
}

// An Engine is a specific piece of a website
// This part handles the login/logout/registration/confirmation pages

type UserEngine struct {
	state pinterface.IUserState
}

func NewUserEngine(userState pinterface.IUserState) (*UserEngine, error) {
	// For the secure cookies
	// This must happen before the random seeding, or
	// else people will have to log in again after every server restart
	web.Config.CookieSecret = cookie.RandomCookieFriendlyString(30)

	rand.Seed(time.Now().UnixNano())

	return &UserEngine{userState}, nil
}

func (ue *UserEngine) GetState() pinterface.IUserState {
	return ue.state
}

// Create a user by adding the username to the list of usernames
func GenerateConfirmUser(state pinterface.IUserState) WebHandle {
	return func(ctx *web.Context, val string) string {
		confirmationCode := val

		unconfirmedUsernames, err := state.AllUnconfirmedUsernames()
		if err != nil {
			return MessageOKurl("Confirmation", "All users are confirmed already.", "/register")
		}

		// Find the username by looking up the confirmationCode on unconfirmed users
		username := ""
		for _, aUsername := range unconfirmedUsernames {
			aConfirmationCode, err := state.ConfirmationCode(aUsername)
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
			return MessageOKurl("Confirmation", "The confirmation link is no longer valid.", "/register")
		}
		hasUser := state.HasUser(username)
		if !hasUser {
			return MessageOKurl("Confirmation", "The user you wish to confirm does not exist anymore.", "/register")
		}

		// Remove from the list of unconfirmed usernames
		state.RemoveUnconfirmed(username)

		// Mark user as confirmed
		state.MarkConfirmed(username)

		return MessageOKurl("Confirmation", "Thank you "+username+", you can now log in.", "/login")
	}
}

// Log in a user by changing the loggedin value
func GenerateLoginUser(state pinterface.IUserState) WebHandle {
	return func(ctx *web.Context, val string) string {
		// Fetch password from ctx
		password, found := ctx.Params["password"]
		if !found {
			return MessageOKback("Login", "Can't log in without a password.")
		}
		username := val
		if username == "" {
			return MessageOKback("Login", "Can't log in with a blank username.")
		}
		if !state.HasUser(username) {
			return MessageOKback("Login", "User "+username+" does not exist, could not log in.")
		}
		if !state.IsConfirmed(username) {
			return MessageOKback("Login", "The email for "+username+" has not been confirmed, check your email and follow the link.")
		}
		if !state.CorrectPassword(username, password) {
			return MessageOKback("Login", "Wrong password.")
		}

		// Log in the user by changing the database and setting a secure cookie
		state.SetLoggedIn(username)

		// Also store the username in the browser
		state.SetUsernameCookie(ctx.ResponseWriter, username)

		// TODO: Use a welcoming messageOK where the user can see when he/she last logged in and from which host

		if username == "admin" {
			ctx.SetHeader("Refresh", "0; url=/admin", true)
		} else {
			// TODO: Redirect to the page the user was at before logging in
			ctx.SetHeader("Refresh", "0; url=/", true)
		}

		return ""
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
func GenerateRegisterUser(state pinterface.IUserState, site string) WebHandle {
	return func(ctx *web.Context, val string) string {

		// Password checks
		password1, found := ctx.Params["password1"]
		if password1 == "" || !found {
			return MessageOKback("Register", "Can't register without a password.")
		}
		password2, found := ctx.Params["password2"]
		if password2 == "" || !found {
			return MessageOKback("Register", "Please confirm the password by typing it in twice.")
		}
		if password1 != password2 {
			return MessageOKback("Register", "The password and confirmation password must be equal.")
		}

		// Email checks
		email, found := ctx.Params["email"]
		if !found {
			return MessageOKback("Register", "Can't register without an email address.")
		}
		// must have @ and ., but no " "
		if !strings.Contains(email, "@") || !strings.Contains(email, ".") || strings.Contains(email, " ") {
			return MessageOKback("Register", "Please use a valid email address.")
		}
		if email != CleanUserInput(email) {
			return MessageOKback("Register", "The sanitized email differs from the given email.")
		}

		// Username checks
		username := val
		if username == "" {
			return MessageOKback("Register", "Can't register without a username.")
		}
		if state.HasUser(username) {
			return MessageOKback("Register", "That user already exists, try another username.")
		}

		// Only some letters are allowed in the username
		err := ValidUsernamePassword(username, password1)
		if err != nil {
			return MessageOKback("Register", err.Error())
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
			return MessageOKurl("Registration complete", "Thanks for registering, the admin user has been created.", "/login")

		}

		// Send confirmation email
		ConfirmationEmail(site, "https://"+site+"/confirm/"+confirmationCode, username, email)

		// Register the need to be confirmed
		state.AddUnconfirmed(username, confirmationCode)

		// Redirect
		return MessageOKurl("Registration complete", "Thanks for registering, the confirmation e-mail has been sent.", "/login")
	}
}

// Log out a user by changing the loggedin value
func GenerateLogoutCurrentUser(state pinterface.IUserState) SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := state.Username(ctx.Request)
		if username == "" {
			return MessageOKback("Logout", "No user to log out")
		}
		if !state.HasUser(username) {
			return MessageOKback("Logout", "user "+username+" does not exist, could not log out")
		}

		// Log out the user by changing the database, the cookie can stay
		state.SetLoggedOut(username)

		// Redirect
		//ctx.SetHeader("Refresh", "0; url=/login", true)
		return MessageOKurl("Logout", username+" is now logged out. Hope to see you soon!", "/login")
	}
}

func GenerateNoJavascriptMessage() SimpleContextHandle {
	return func(ctx *web.Context) string {
		return MessageOKback("JavaScript error", "Cookies and Javascript must be enabled.<br />Older browsers might be supported in the future.")
	}
}

func LoginCP(basecp BaseCP, state pinterface.IUserState, url string) *ContentPage {
	cp := basecp(state)
	cp.ContentTitle = "Login"
	cp.ContentHTML = LoginForm()
	cp.ContentJS += OnClick("#loginButton", "$('#loginForm').get(0).setAttribute('action', '/login/' + $('#username').val());")
	//cp.ExtraCSSurls = append(cp.ExtraCSSurls, "/css/login.css")
	cp.Url = url

	// Hide the Login menu if we're on the Login page
	//cp.HeaderJS = strings.Replace(cp.HeaderJS, "menuLogin", "menuNop", 1)
	//cp.ContentJS += Hide("#menuLogin")

	return cp
}

func RegisterCP(basecp BaseCP, state pinterface.IUserState, url string) *ContentPage {
	cp := basecp(state)
	cp.ContentTitle = "Register"
	cp.ContentHTML = RegisterForm()
	cp.ContentJS += OnClick("#registerButton", "$('#registerForm').get(0).setAttribute('action', '/register/' + $('#username').val());")
	//cp.ExtraCSSurls = append(cp.ExtraCSSurls, "/css/register.css")
	cp.Url = url

	// Hide the Register menu if we're on the Register page
	//cp.HeaderJS = strings.Replace(cp.HeaderJS, "menuRegister", "menuNop", 1)
	//cp.ContentJS += Hide("#menuRegister")

	return cp
}

// Site is ie. "archlinux.no" and used for sending confirmation emails
func (ue *UserEngine) ServePages(site string) {
	state := ue.state
	web.Post("/register/(.*)", GenerateRegisterUser(state, site))
	web.Post("/register", GenerateNoJavascriptMessage())
	web.Post("/login/(.*)", GenerateLoginUser(state))
	web.Post("/login", GenerateNoJavascriptMessage())
	web.Get("/logout", GenerateLogoutCurrentUser(state))
	web.Get("/confirm/(.*)", GenerateConfirmUser(state))
}
