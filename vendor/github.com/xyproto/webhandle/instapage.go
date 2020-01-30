package webhandle

// Login form, registration form, "message box" and other "one off" webpages.

// HTML page that just displays a message
func Message(title, msg string) string {
	return "<!DOCTYPE html><html><head><title>" + title + "</title></head><body style=\"margin:4em; font-family:courier; color:gray; background-color:light gray;\"><h2>" + title + "</h2><hr style=\"margin-top:-1em; margin-bottom: 2em; margin-right: 20%; text-align: left; border: 1px dotted #b0b0b0; height:1px;\"><div style=\"margin-left: 2em;\">" + msg + "</div></body></html>"
}

// HTML page that redirects to an url by using JavaScript
func HTMLPageRedirect(url string) string {
	return "<!DOCTYPE html><html><head><script type=\"text/javascript\">window.location.href = \"" + url + "\";</script></head></html>"
}

// Generic login form, not a complete html page.
// Sends a POST request to /login. Passes "password". (Only "name=" fields).
// Remember to don't send passwords in plaintext. Use https.
func LoginForm() string {
	return "<form id=\"loginForm\" action=\"/login\" method=\"POST\"><div style=\"margin: 1em;\"><label for=\"username\" style=\"display: inline-block; float: left; clear: left; width: 150px; text-align: right; margin-right: 2em;\">Username:</label><input style=\"display:inline-block; float:left;\" id=\"username\"><br /><label for=\"password\" style=\"display: inline-block; float: left; clear: left; width: 150px; text-align: right; margin-right: 2em;\">Password:</label><input style=\"display:inline-block; float:left;\" id=\"password\" type=\"password\" name=\"password\"></div><br /><p><button style=\"font-size: 1.5em; margin-left: 10em; width: 6em; height: 2.2em; margin-top: 0px; border: 2px solid black; background-color: #3ba0d8; border-radius:10px/6px;\" id=\"loginButton\">Login</button></p><script type=\"text/javascript\">document.getElementById(\"username\").focus();</script>"
}

// Generic registration form, not a complete html page.
// Sends a POST request to /register. Passes "password1", "password2" and "email". (Only "name=" fields).
// Remember to don't send passwords in plaintext. Use https.
func RegisterForm() string {
	return "<form id=\"registerForm\" action=\"/register\" method=\"POST\"><div style=\"margin: 1em;\"><label for=\"username\" style=\"display: inline-block; float: left; clear: left; width: 150px; text-align: right; margin-right: 2em;\">Username:</label><input style=\"display:inline-block; float:left;\" id=\"username\"><br /><label for=\"password1\" style=\"display: inline-block; float: left; clear: left; width: 150px; text-align: right; margin-right: 2em;\">Password:</label><input style=\"display:inline-block; float:left;\" id=\"password1\" type=\"password\" name=\"password1\"><br /><label for=\"password2\" style=\"display: inline-block; float: left; clear: left; width: 150px; text-align: right; margin-right: 2em;\">Confirm password:</label><input style=\"display:inline-block; float:left;\" id=\"password2\" type=\"password\" name=\"password2\"><br /><label for=\"email\" style=\"display: inline-block; float: left; clear: left; width: 150px; text-align: right; margin-right: 2em;\">Email:</label><input name=\"email\" style=\"display:inline-block; float:left;\" id=\"email\"></div><br /><p><button style=\"font-size: 1.5em; margin-left: 10em; width: 6em; height: 2.2em; margin-top: 0.2em; border: 2px solid black; background-color: #50d080; border-radius:10px/6px;\" id=\"registerButton\">Register</button></p></form><script type=\"text/javascript\">document.getElementById(\"username\").focus();</script>"

}

// Generic HTML page for displaying a message and a button that takes the user somewhere else by using JavaScript
func messageComposer(title, msg, javascript string) string {
	return "<!DOCTYPE html><html><head><title>" + title + "</title></head><body style=\"margin:4em; font-family:courier; color:#101010; background-color:#e0e0e0;\"><h2>" + title + "</h2><hr style=\"margin-top:-1em; margin-bottom: 2em; margin-right: 20%; text-align: left; border: 1px dotted #202020; height:1px;\"><div style=\"margin-left: 2em;\">" + msg + "<br /><br /><button id=\"okbutton\" style=\"margin-top: 2em; margin-left: 20em;\" onclick=\"" + javascript + "\">OK</button></div><script type=\"text/javascript\">document.getElementById(\"okbutton\").focus();</script></body></html>"
}

// Message page where the ok button goes one back in history by using JavaScript
func MessageOKback(title, msg string) string {
	return messageComposer(title, msg, "history.go(-1);")
}

// Message page where the ok button goes to a given url using JavaScript
func MessageOKurl(title, msg, url string) string {
	return messageComposer(title, msg, "location.href='"+url+"';")
}

// Button for going back in history
func BackButton() string {
	return "<button onClick='history.go(-1);'>Back</button>"
}
