package siteengines

// TODO: Check for the visual likeness of two usernames when checking for availability! Generate images and compare pixels.
// TODO: Logging in should work case sensitively, or at least without concern for the case of the first letter
// TODO: Consider using "0" and "1" instead of "true" or "false" when setting values, while still understanding "true" or "false"
// TODO: The password should be set at confirmation time instead of registration-time in order to make the process clearer and pave the way for invite-only?

import (
	"math/rand"
	"strings"
	"time"

	"github.com/hoisie/web"
	. "github.com/xyproto/browserspeak"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/instapage"
)

// An Engine is a specific piece of a website
// This part handles the login/logout/registration/confirmation pages

const (
	MINIMUM_CONFIRMATION_CODE_LENGTH = 20
	USERNAME_ALLOWED_LETTERS         = "abcdefghijklmnopqrstuvwxyzæøåABCDEFGHIJKLMNOPQRSTUVWXYZÆØÅ_0123456789"
)

type UserEngine struct {
	state *UserState
}

func NewUserEngine(userState *UserState) *UserEngine {
	// For the secure cookies
	// This must happen before the random seeding, or
	// else people will have to log in again after every server restart
	web.Config.CookieSecret = RandomCookieFriendlyString(30)

	rand.Seed(time.Now().UnixNano())

	return &UserEngine{userState}
}

func (ue *UserEngine) GetState() *UserState {
	return ue.state
}

// Create a user by adding the username to the list of usernames
func GenerateConfirmUser(state *UserState) WebHandle {
	return func(ctx *web.Context, val string) string {
		confirmationCode := val

		unconfirmedUsernames, err := state.GetAllUnconfirmedUsernames()
		if err != nil {
			return instapage.MessageOKurl("Confirmation", "All users are confirmed already.", "/register")
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
			return instapage.MessageOKurl("Confirmation", "The confirmation link is no longer valid.", "/register")
		}
		hasUser := state.HasUser(username)
		if !hasUser {
			return instapage.MessageOKurl("Confirmation", "The user you wish to confirm does not exist anymore.", "/register")
		}

		// Remove from the list of unconfirmed usernames
		state.RemoveUnconfirmed(username)

		// Mark user as confirmed
		state.MarkConfirmed(username)

		return instapage.MessageOKurl("Confirmation", "Thank you "+username+", you can now log in.", "/login")
	}
}

// Log in a user by changing the loggedin value
func GenerateLoginUser(state *UserState) WebHandle {
	return func(ctx *web.Context, val string) string {
		// Fetch password from ctx
		password, found := ctx.Params["password"]
		if !found {
			return instapage.MessageOKback("Login", "Can't log in without a password.")
		}
		username := val
		if username == "" {
			return instapage.MessageOKback("Login", "Can't log in with a blank username.")
		}
		if !state.HasUser(username) {
			return instapage.MessageOKback("Login", "User "+username+" does not exist, could not log in.")
		}
		if !state.IsConfirmed(username) {
			return instapage.MessageOKback("Login", "The email for "+username+" has not been confirmed, check your email and follow the link.")
		}
		if !CorrectPassword(state, username, password) {
			return instapage.MessageOKback("Login", "Wrong password.")
		}

		// Log in the user by changing the database and setting a secure cookie
		state.SetLoggedIn(username)

		state.SetBrowserUsername(ctx, username)

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
func GenerateRegisterUser(state *UserState, site string) WebHandle {
	return func(ctx *web.Context, val string) string {

		// Password checks
		password1, found := ctx.Params["password1"]
		if password1 == "" || !found {
			return instapage.MessageOKback("Register", "Can't register without a password.")
		}
		password2, found := ctx.Params["password2"]
		if password2 == "" || !found {
			return instapage.MessageOKback("Register", "Please confirm the password by typing it in twice.")
		}
		if password1 != password2 {
			return instapage.MessageOKback("Register", "The password and confirmation password must be equal.")
		}

		// Email checks
		email, found := ctx.Params["email"]
		if !found {
			return instapage.MessageOKback("Register", "Can't register without an email address.")
		}
		// must have @ and ., but no " "
		if !strings.Contains(email, "@") || !strings.Contains(email, ".") || strings.Contains(email, " ") {
			return instapage.MessageOKback("Register", "Please use a valid email address.")
		}
		if email != CleanUserInput(email) {
			return instapage.MessageOKback("Register", "The sanitized email differs from the given email.")
		}

		// Username checks
		username := val
		if username == "" {
			return instapage.MessageOKback("Register", "Can't register without a username.")
		}
		if state.HasUser(username) {
			return instapage.MessageOKback("Register", "That user already exists, try another username.")
		}

		// Only some letters are allowed in the username
	NEXT:
		for _, letter := range username {
			for _, allowedLetter := range USERNAME_ALLOWED_LETTERS {
				if letter == allowedLetter {
					continue NEXT
				}
			}
			return instapage.MessageOKback("Register", "Only a-å, A-Å, 0-9 and _ are allowed in usernames.")
		}
		if username == password1 {
			return instapage.MessageOKback("Register", "Username and password must be different, try another password.")
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

		// The confirmation code must be a minimum of 8 letters long
		length := MINIMUM_CONFIRMATION_CODE_LENGTH
		confirmationCode := RandomHumanFriendlyString(length)
		for AlreadyHasConfirmationCode(state, confirmationCode) {
			// Increase the length of the confirmationCode random string every time there is a collision
			length++
			confirmationCode = RandomHumanFriendlyString(length)
			if length > 100 {
				// This should never happen
				panic("ERROR: Too many generated confirmation codes are not unique, something is wrong")
			}
		}

		// Send confirmation email
		ConfirmationEmail(site, "https://"+site+"/confirm/"+confirmationCode, username, email)

		// Register the need to be confirmed
		state.AddUnconfirmed(username, confirmationCode)

		// Redirect
		return instapage.MessageOKurl("Registration complete", "Thanks for registering, the confirmation e-mail has been sent.", "/login")
	}
}

// Log out a user by changing the loggedin value
func GenerateLogoutCurrentUser(state *UserState) SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := GetBrowserUsername(ctx)
		if username == "" {
			return instapage.MessageOKback("Logout", "No user to log out")
		}
		if !state.HasUser(username) {
			return instapage.MessageOKback("Logout", "user "+username+" does not exist, could not log out")
		}

		// Log out the user by changing the database, the cookie can stay
		state.SetLoggedOut(username)

		// Redirect
		//ctx.SetHeader("Refresh", "0; url=/login", true)
		return instapage.MessageOKurl("Logout", username+" is now logged out. Hope to see you soon!", "/login")
	}
}

func GenerateNoJavascriptMessage() SimpleContextHandle {
	return func(ctx *web.Context) string {
		return instapage.MessageOKback("JavaScript error", "Cookies and Javascriåt must be enabled.<br />Older browsers might be supported in the future.")
	}
}

func LoginCP(basecp BaseCP, state *UserState, url string) *ContentPage {
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

func RegisterCP(basecp BaseCP, state *UserState, url string) *ContentPage {
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
func (ue *UserEngine) ServePages(site string) {
	state := ue.state
	web.Post("/register/(.*)", GenerateRegisterUser(state, site))
	web.Post("/login/(.*)", GenerateLoginUser(state))
	web.Post("/login", GenerateNoJavascriptMessage())
	web.Get("/logout", GenerateLogoutCurrentUser(state))
	web.Get("/confirm/(.*)", GenerateConfirmUser(state))
}
