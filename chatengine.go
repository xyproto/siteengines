package main

import (
	"strconv"
	"time"

	. "github.com/xyproto/browserspeak"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/instapage"
	. "github.com/xyproto/simpleredis"
	"github.com/xyproto/web"
)

// An Engine is a specific piece of a website
// This part handles the "chat" pages

type ChatEngine struct {
	userState *UserState
	chatState *ChatState
}

type ChatState struct {
	active   *RedisSet       // A list of all users that are in the chat, must correspond to the users in UserState.users
	said     *RedisList      // A list of everything that has been said so far
	userInfo *RedisHashMap   // Info about a chat user - last seen, preferred number of lines etc
	pool     *ConnectionPool // A connection pool for Redis
}

func NewChatEngine(userState *UserState) *ChatEngine {
	pool := userState.GetPool()
	chatState := new(ChatState)
	chatState.active = NewRedisSet(pool, "active")
	chatState.said = NewRedisList(pool, "said")
	chatState.userInfo = NewRedisHashMap(pool, "userInfo") // lastSeen.time is an encoded timestamp for when the user was last seen chatting
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
		return "BANANAS!"
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

		retval := "Hi " + username + "<br />"
		retval += "<br />"
		retval += "Participants:" + "<br />"
		// TODO: If the person has not been seen the last 96 hours, don't list him/her
		for _, otherUser := range ce.GetChatUsers() {
			if otherUser == username {
				continue
			}
			retval += "&nbsp;&nbsp;" + otherUser + ", last seen " + ce.GetLastSeen(otherUser) + "<br />"
		}
		retval += "<br />"
		retval += "<div style='background-color: white; padding: 1em;'>"
		retval += ce.chatText(ce.GetLines(username))
		retval += "</div>"
		retval += "<br />"
		// The say() function for submitting text over ajax (a post request), clearing the text intput field and updating the chat text
		retval += JS("function say(text) { $.post('/say', {said:$('#sayText').val()}, function(data) { $('#sayText').val(''); $('#chatText').html(data); }); }")
		// Call say() at return 
		retval += "<input size='60' id='sayText' name='said' type='text' onKeypress=\"if (event.keyCode == 13) { say($('#sayText').val()); };\">"
		// Cal say() at the click of the button
		retval += "<button onClick='say();'>Say</button>"
		// Focus on the text input
		retval += JS(Focus("#sayText"))
		// TODO: Update the chat every 64 seconds. If something happens, update every 200ms, then 400ms, then 800ms etc until it's at 64 seconds again. This should happen in javascript.
		// Update the chat text every 500 ms
		retval += JS("setInterval(function(){$.post('/say', {}, function(data) { $('#chatText').html(data); });}, 500);")
		// A function for setting the preferred number of lines
		retval += JS("function setlines(numlines) { $.post('/setchatlines', {lines:numlines}, function(data) { $('#chatText').html(data); }); }")
		// A button for viewing 20 lines at a time
		retval += "<button onClick='setlines(20);'>20</button>"
		// A button for viewing 50 lines at a time
		retval += "<button onClick='setlines(50);'>50</button>"
		// A button for viewing all lines at a time
		retval += "<button onClick='setlines(-1);'>all</button>"
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

		ce.Say(username, CleanUpUserInput(said))

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
