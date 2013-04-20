package siteengines

import (
	"strconv"
	"time"

	. "github.com/xyproto/browserspeak"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/instapage"
	"github.com/xyproto/simpleredis"
	"github.com/xyproto/web"
)

// An Engine is a specific piece of a website
// This part handles the "chat" pages

type ChatEngine struct {
	userState *UserState
	chatState *ChatState
}

type ChatState struct {
	active   *simpleredis.Set            // A list of all users that are in the chat, must correspond to the users in UserState.users
	said     *simpleredis.List           // A list of everything that has been said so far
	userInfo *simpleredis.HashMap        // Info about a chat user - last seen, preferred number of lines etc
	pool     *simpleredis.ConnectionPool // A connection pool for Redis
}

func NewChatEngine(userState *UserState) *ChatEngine {
	pool := userState.GetPool()
	chatState := new(ChatState)
	chatState.active = simpleredis.NewSet(pool, "active")
	chatState.said = simpleredis.NewList(pool, "said")
	chatState.userInfo = simpleredis.NewHashMap(pool, "userInfo") // lastSeen.time is an encoded timestamp for when the user was last seen chatting
	chatState.pool = pool
	return &ChatEngine{userState, chatState}
}

func (ce *ChatEngine) ServePages(basecp BaseCP, menuEntries MenuEntries) {
	chatCP := basecp(ce.userState)
	chatCP.ContentTitle = "Chat"
	chatCP.ExtraCSSurls = append(chatCP.ExtraCSSurls, "/css/chat.css")

	tvgf := DynamicMenuFactoryGenerator(menuEntries)
	tvg := tvgf(ce.userState)

	web.Get("/chat", chatCP.WrapSimpleContextHandle(ce.GenerateChatCurrentUser(), tvg))
	web.Post("/say", ce.GenerateSayCurrentUser())
	web.Get("/css/chat.css", ce.GenerateCSS(chatCP.ColorScheme))
	web.Post("/setchatlines", ce.GenerateSetChatLinesCurrentUser())
	// For debugging
	web.Get("/getchatlines", ce.GenerateGetChatLinesCurrentUser())
}

func (ce *ChatEngine) SetLines(username string, lines int) {
	ce.chatState.userInfo.Set(username, "lines", strconv.Itoa(lines))
}

func (ce *ChatEngine) GetLines(username string) int {
	val, err := ce.chatState.userInfo.Get(username, "lines")
	if err != nil {
		// The default
		return 20
	}
	num, err := strconv.Atoi(val)
	if err != nil {
		// The default
		return 20
	}
	return num
}

// Mark a user as seen
func (ce *ChatEngine) Seen(username string) {
	now := time.Now()
	encodedTime, err := now.GobEncode()
	if err != nil {
		panic("ERROR: Can't encode the time")
	}
	ce.chatState.userInfo.Set(username, "lastseen", string(encodedTime))
}

// Checks if the user has been seen lately (within 12 hours ago)
func (ce *ChatEngine) SeenLately(username string) bool {
	encodedTime, err := ce.chatState.userInfo.Get(username, "lastseen")
	if err != nil {
		return false
	}
	var then time.Time
	err = then.GobDecode([]byte(encodedTime))
	if err != nil {
		return false
	}
	notTooLongDuration, err := time.ParseDuration("-12h")
	if err != nil {
		return false
	}
	notTooLongAgo := time.Now().Add(notTooLongDuration)
	if then.After(notTooLongAgo) {
		return true
	}
	return false
}

func (ce *ChatEngine) GetLastSeen(username string) string {
	encodedTime, err := ce.chatState.userInfo.Get(username, "lastseen")
	if err == nil {
		var then time.Time
		err = then.GobDecode([]byte(encodedTime))
		if err == nil {
			timestamp := then.String()
			return timestamp[11:19]
		}
	}
	return "never"
}

func (ce *ChatEngine) IsChatting(username string) bool {
	encodedTime, err := ce.chatState.userInfo.Get(username, "lastseen")
	if err == nil {
		var then time.Time
		err = then.GobDecode([]byte(encodedTime))
		if err == nil {
			elapsed := time.Since(then)
			if elapsed.Minutes() > 20 {
				// 20 minutes since last seen saying anything, set as not chatting
				ce.SetChatting(username, false)
				return false
			}
		}
	}
	// TODO: If the user was last seen more than N minutes ago, set as not chatting and return false
	return ce.userState.GetBooleanField(username, "chatting")
}

// Set "chatting" to "true" or "false" for a given user
func (ce *ChatEngine) SetChatting(username string, val bool) {
	ce.userState.SetBooleanField(username, "chatting", val)
}

func (ce *ChatEngine) JoinChat(username string) {
	// Join the chat
	ce.chatState.active.Add(username)
	// Change the chat status for the user
	ce.SetChatting(username, true)
	// Mark the user as seen
	ce.Seen(username)
}

func (ce *ChatEngine) Say(username, text string) {
	timestamp := time.Now().String()
	textline := timestamp[11:19] + "&nbsp;&nbsp;" + username + "> " + text
	ce.chatState.said.Add(textline)
	// Store the timestamp for when the user was last seen as well
	ce.Seen(username)
}

func LeaveChat(ce *ChatEngine, username string) {
	// Leave the chat
	ce.chatState.active.Del(username)
	// Change the chat status for the user
	ce.SetChatting(username, false)
}

func (ce *ChatEngine) GetChatUsers() []string {
	chatUsernames, err := ce.chatState.active.GetAll()
	if err != nil {
		return []string{}
	}
	return chatUsernames
}

func (ce *ChatEngine) GetChatText() []string {
	chatText, err := ce.chatState.said.GetAll()
	if err != nil {
		return []string{}
	}
	return chatText
}

// Get the last N entries
func (ce *ChatEngine) GetLastChatText(n int) []string {
	chatText, err := ce.chatState.said.GetLastN(n)
	if err != nil {
		return []string{}
	}
	return chatText
}

func (ce *ChatEngine) chatText(lines int) string {
	if lines == -1 {
		return ""
	}
	retval := "<div id='chatText'>"
	// Show N lines of chat text
	for _, said := range ce.GetLastChatText(lines) {
		retval += said + "<br />"
	}
	return retval + "</div>"
}

func (ce *ChatEngine) GenerateChatCurrentUser() SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := GetBrowserUsername(ctx)
		if username == "" {
			return "No user logged in"
		}
		if !ce.userState.IsLoggedIn(username) {
			return "Not logged in"
		}

		ce.JoinChat(username)

		// TODO: Add a button for someone to see the entire chat
		// TODO: Add some protection against random monkeys that only fling poo

		retval := "<h2>Hi " + username + "</h2>"
		seenusers := ""
		for _, otherUser := range ce.GetChatUsers() {
			if otherUser == username {
				continue
			}
			if ce.SeenLately(otherUser) {
				seenusers += "&nbsp;&nbsp;" + otherUser + ", last seen " + ce.GetLastSeen(otherUser) + "<br />"
			}
		}
		// Add a list of participants that has been seen lately, if there are any
		if seenusers != "" {
			retval += "<br />Other participants:<br />"
			retval += seenusers
			retval += "<br />"
		}
		retval += "<div style='background-color: white; padding: 1em;'>"
		retval += ce.chatText(ce.GetLines(username))
		retval += "</div>"
		retval += "<br />"
		retval += JS("var fastestPolling = 500;")
		retval += JS("var slowestPolling = 64000;")
		retval += JS("var pollInterval = fastestPolling;")
		retval += JS("var inactivityCounter = 0;")
		retval += JS("var inactivityTimeout = 20;") // Chat times out after 20 periods of slowest polling (approximately 20 minutes)
		retval += JS("var pollID = 0;")
		// The say() function for submitting text over ajax (a post request), clearing the text intput field and updating the chat text.
		// Also sets the polling interval to the fastest value.
		retval += JS(`function say(text) {
			inactivityCounter = 0;
			pollInterval = fastestPolling;
			$.post('/say', {said:$('#sayText').val()}, function(data) { $('#sayText').val(''); $('#chatText').html(data); });
		}`)
		// Call say() at return 
		retval += "<input size='60' id='sayText' name='said' type='text' onKeypress=\"if (event.keyCode == 13) { say($('#sayText').val()); };\">"
		// Cal say() at the click of the button
		retval += "<button onClick='say();'>Say</button>"
		// Focus on the text input
		retval += JS(Focus("#sayText"))
		// Update the chat text. Reduce the poll interval at every poll.
		// When the user does something, the polling interval will be reset to something quicker.
		retval += JS(`function UpdateChat() {
		    if (pollInterval < slowestPolling) {
			    pollInterval *= 2;
				clearInterval(pollID);
				pollID = setInterval(UpdateChat, pollInterval);
			} else {
				inactivityCounter++;
			}
			if inactivityCounter < inactivityTimeout {
				$.post('/say', {}, function(data) { $('#chatText').html(data); });
			}
		}`)
		retval += JS("pollID = setInterval(UpdateChat, pollInterval);")
		// A function for setting the preferred number of lines
		retval += JS("function setlines(numlines) { $.post('/setchatlines', {lines:numlines}, function(data) { $('#chatText').html(data); " + ScrollDownAnimated() + "}); }")
		// A button for viewing 20 lines at a time
		retval += "<button onClick='setlines(20);'>20</button>"
		// A button for viewing 50 lines at a time
		retval += "<button onClick='setlines(50);'>50</button>"
		// A button for viewing 99999 lines at a time
		retval += "<button onClick='setlines(99999);'>99999</button>"
		// For viewing all the text so far

		return retval
	}
}

func (ce *ChatEngine) GenerateSayCurrentUser() SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := GetBrowserUsername(ctx)
		if username == "" {
			return "No user logged in"
		}
		if !ce.userState.IsLoggedIn(username) {
			return "Not logged in"
		}
		if !ce.IsChatting(username) {
			return "Not currently chatting"
		}
		said, found := ctx.Params["said"]
		if !found || said == "" {
			// Return the text instead of giving an error for easy use of /say to refresh the content
			// Note that as long as Say below isn't called, the user will be marked as inactive eventually
			return ce.chatText(ce.GetLines(username))
		}

		ce.Say(username, CleanUserInput(said))

		return ce.chatText(ce.GetLines(username))
	}
}

func (ce *ChatEngine) GenerateGetChatLinesCurrentUser() SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := GetBrowserUsername(ctx)
		if username == "" {
			return "No user logged in"
		}
		if !ce.userState.IsLoggedIn(username) {
			return "Not logged in"
		}
		if !ce.IsChatting(username) {
			return "Not currently chatting"
		}
		num := ce.GetLines(username)

		return strconv.Itoa(num)
	}
}

func (ce *ChatEngine) GenerateSetChatLinesCurrentUser() SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := GetBrowserUsername(ctx)
		if username == "" {
			return "No user logged in"
		}
		if !ce.userState.IsLoggedIn(username) {
			return "Not logged in"
		}
		if !ce.IsChatting(username) {
			return "Not currently chatting"
		}
		lines, found := ctx.Params["lines"]
		if !found || lines == "" {
			return instapage.MessageOKback("Set chat lines", "Missing value for preferred number of lines")
		}
		num, err := strconv.Atoi(lines)
		if err != nil {
			return instapage.MessageOKback("Set chat lines", "Invalid number of lines: "+lines)
		}

		// Set the preferred number of lines for this user
		ce.SetLines(username, num)

		return ce.chatText(num)
	}
}

func (ce *ChatEngine) GenerateCSS(cs *ColorScheme) SimpleContextHandle {
	return func(ctx *web.Context) string {
		ctx.ContentType("css")
		return `
.yes {
	background-color: #90ff90;
	color: black;
}
.no {
	background-color: #ff9090;
	color: black;
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

#chatText {
	background-color: white;
}

`
		//
	}
}
