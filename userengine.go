package genericsite

// TODO: Check for the visual likeness of two usernames when checking for availability! Generate images and compare pixels.
// TODO: Logging in should work case sensitively, or at least without concern for the case of the first letter

// TODO: Consider using "0" and "1" instead of "true" or "false" when setting values, while still understanding "true" or "false"

import (
	"crypto/sha256"
	"errors"
	"io"
	"math/rand"
	"strings"
	"time"

	. "github.com/xyproto/browserspeak"
	"github.com/xyproto/instapage"
	. "github.com/xyproto/simpleredis"
	"github.com/xyproto/web"
)

type UserState struct {
	// see: http://redis.io/topics/data-types
	users       *RedisHashMap   // Hash map of users, with several different fields per user ("loggedin", "confirmed", "email" etc)
	usernames   *RedisSet       // A list of all usernames, for easy enumeration
	unconfirmed *RedisSet       // A list of unconfirmed usernames, for easy enumeration
	pool        *ConnectionPool // A connection pool for Redis
}

// An Engine is a specific piece of a website
// This part handles the login/logout/registration/confirmation pages

const (
	MINIMUM_CONFIRMATION_CODE_LENGTH = 20
	USERNAME_ALLOWED_LETTERS         = "abcdefghijklmnopqrstuvwxyzæøåABCDEFGHIJKLMNOPQRSTUVWXYZÆØÅ_0123456789"
)

type UserEngine struct {
	state *UserState
}

func NewUserEngine(pool *ConnectionPool) *UserEngine {
	// For the secure cookies
	// This must happen before the random seeding, or
	// else people will have to log in again after every server restart
	web.Config.CookieSecret = RandomCookieFriendlyString(30)

	rand.Seed(time.Now().UnixNano())

	userState := createUserState(pool)
	return &UserEngine{userState}
}

func (state *UserState) GetPool() *ConnectionPool {
	return state.pool
}

func (ue *UserEngine) GetState() *UserState {
	return ue.state
}

// Checks if the current user is logged in as a user right now
func (state *UserState) UserRights(ctx *web.Context) bool {
	if username := GetBrowserUsername(ctx); username != "" {
		return state.IsLoggedIn(username)
	}
	return false
}

func (state *UserState) HasUser(username string) bool {
	val, err := state.usernames.Has(username)
	if err != nil {
		// This happened at concurrent connections before introducing the connection pool
		panic("ERROR: Lost connection to Redis?")
	}
	return val
}

// Creates a user without doing ANY checks
func AddUserUnchecked(state *UserState, username, password, email string) {
	// Add the user
	state.usernames.Add(username)

	// Add password and email
	state.users.Set(username, "password", password)
	state.users.Set(username, "email", email)

	// Addditional fields
	additionalfields := []string{"loggedin", "confirmed", "admin"}
	for _, fieldname := range additionalfields {
		state.users.Set(username, fieldname, "false")
	}
}

func (state *UserState) GetBooleanField(username, fieldname string) bool {
	hasUser := state.HasUser(username)
	if !hasUser {
		return false
	}
	chatting, err := state.users.Get(username, fieldname)
	if err != nil {
		return false
	}
	return TruthValue(chatting)
}

func (state *UserState) SetBooleanField(username, fieldname string, val bool) {
	strval := "false"
	if val {
		strval = "true"
	}
	state.users.Set(username, fieldname, strval)
}

func (state *UserState) IsConfirmed(username string) bool {
	return state.GetBooleanField(username, "confirmed")
}

func CorrectPassword(state *UserState, username, password string) bool {
	hashedPassword, err := state.users.Get(username, "password")
	if err != nil {
		return false
	}
	if hashedPassword == HashPasswordVersion2(password) {
		return true
	}
	return false
}

func (state *UserState) GetConfirmationCode(username string) string {
	confirmationCode, err := state.users.Get(username, "confirmationCode")
	if err != nil {
		return ""
	}
	return confirmationCode
}

// Goes through all the confirmationCodes of all the unconfirmed users
// and checks if this confirmationCode already is in use
func AlreadyHasConfirmationCode(state *UserState, confirmationCode string) bool {
	unconfirmedUsernames, err := state.unconfirmed.GetAll()
	if err != nil {
		return false
	}
	for _, aUsername := range unconfirmedUsernames {
		aConfirmationCode, err := state.users.Get(aUsername, "confirmationCode")
		if err != nil {
			// TODO: Consider just logging the incident instead
			panic("ERROR: Inconsistent user")
			//continue
		}
		if confirmationCode == aConfirmationCode {
			// Found it
			return true
		}
	}
	return false
}

// Create a user by adding the username to the list of usernames
func GenerateConfirmUser(state *UserState) WebHandle {
	return func(ctx *web.Context, val string) string {
		confirmationCode := val

		unconfirmedUsernames, err := state.unconfirmed.GetAll()
		if err != nil {
			return instapage.MessageOKurl("Confirmation", "All users are confirmed already.", "/register")
		}

		// Find the username by looking up the confirmationCode on unconfirmed users
		username := ""
		for _, aUsername := range unconfirmedUsernames {
			aSecret, err := state.users.Get(aUsername, "confirmationCode")
			if err != nil {
				// TODO: Inconsistent user! Log this.
				continue
			}
			if confirmationCode == aSecret {
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
		state.unconfirmed.Del(username)
		// Remove the confirmationCode from the user
		state.users.Del(username, "confirmationCode")

		// Mark user as confirmed
		state.users.Set(username, "confirmed", "true")

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
		state.users.Set(username, "loggedin", "true")

		// TODO: Users should be able to select their own cookie timeout
		state.SetBrowserUsername(ctx, username, 3600)

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

func HashPasswordVersion2(password string) string {
	hasher := sha256.New()
	// TODO: Read up on password hashing
	io.WriteString(hasher, password+"some salt is better than none")
	return string(hasher.Sum(nil))
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

// Register a new user
func GenerateRegisterUser(state *UserState) WebHandle {
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
		password := HashPasswordVersion2(password1)
		AddUserUnchecked(state, username, password, email)

		// Mark user as administrator if that is the case
		if adminuser {
			// This does not set the username to admin,
			// but sets the admin field to true
			state.users.Set(username, "admin", "true")
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
		ConfirmationEmail("archlinux.no", "https://archlinux.no/confirm/"+confirmationCode, username, email)

		// Register the need to be confirmed
		state.unconfirmed.Add(username)
		state.users.Set(username, "confirmationCode", confirmationCode)

		// Redirect
		//ctx.SetHeader("Refresh", "0; url=/login", true)
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
		state.users.Set(username, "loggedin", "false")

		// Redirect
		//ctx.SetHeader("Refresh", "0; url=/login", true)
		return instapage.MessageOKurl("Logout", username+" is now logged out. Hope to see you soon!", "/login")
	}
}

// Checks if the given username is logged in or not
func (state *UserState) IsLoggedIn(username string) bool {
	if !state.HasUser(username) {
		return false
	}
	status, err := state.users.Get(username, "loggedin")
	if err != nil {
		// Returns "no" if the status can not be retrieved
		return false
	}
	return TruthValue(status)
}

// Gets the username that is stored in a cookie in the browser, if available
func GetBrowserUsername(ctx *web.Context) string {
	username, _ := ctx.GetSecureCookie("user")
	// TODO: Return err, then the calling function should notify the user that cookies are needed
	return username
}

func (state *UserState) SetBrowserUsername(ctx *web.Context, username string, timeout int64) error {
	if username == "" {
		return errors.New("Can't set cookie for empty username")
	}
	if !state.HasUser(username) {
		return errors.New("Can't store cookie for non-existsing user")
	}
	// Create a cookie that lasts for a while ("timeout" seconds),
	// this is the equivivalent of a session for a given username.
	ctx.SetSecureCookiePath("user", username, timeout, "/")
	return nil
}

func GenerateNoJavascriptMessage() SimpleContextHandle {
	return func(ctx *web.Context) string {
		return instapage.MessageOKback("JavaScript error", "Cookies and Javascriåt must be enabled.<br />Older browsers might be supported in the future.")
	}
}

func createUserState(pool *ConnectionPool) *UserState {
	// For the database
	state := new(UserState)
	state.users = NewRedisHashMap(pool, "users")
	state.usernames = NewRedisSet(pool, "usernames")
	state.unconfirmed = NewRedisSet(pool, "unconfirmed")
	state.pool = pool
	return state
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

func (ue *UserEngine) ServeSystem() {
	state := ue.state
	web.Post("/register/(.*)", GenerateRegisterUser(state))
	web.Post("/login/(.*)", GenerateLoginUser(state))
	web.Post("/login", GenerateNoJavascriptMessage())
	web.Get("/logout", GenerateLogoutCurrentUser(state))
	web.Get("/confirm/(.*)", GenerateConfirmUser(state))
}
