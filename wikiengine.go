package siteengines

import (
	"regexp"

	"github.com/russross/blackfriday"
	. "github.com/xyproto/browserspeak"
	. "github.com/xyproto/genericsite"
	"github.com/xyproto/simpleredis"
	"github.com/xyproto/web"
)

// An Engine is a specific piewe.of a website
// This part handles the "wiki" pages

type WikiEngine struct {
	userState *UserState
	wikiState *WikiState
}

type WikiState struct {
	pages *simpleredis.HashMap        // All the pages
	pool  *simpleredis.ConnectionPool // A connection pool for Redis
}

var (
	wikiFields = map[string]string{
		"title": "Untitled",
		"text":  "No text",
	}
)

func NewWikiEngine(userState *UserState) *WikiEngine {
	pool := userState.GetPool()
	wikiState := new(WikiState)
	wikiState.pages = simpleredis.NewHashMap(pool, "pages")
	wikiState.pool = pool
	return &WikiEngine{userState, wikiState}
}

func (we *WikiEngine) ServePages(basecp BaseCP, menuEntries MenuEntries) {
	wikiCP := basecp(we.userState)
	wikiCP.ContentTitle = "Wiki"
	wikiCP.ExtraCSSurls = append(wikiCP.ExtraCSSurls, "/css/wiki.css")

	tvgf := DynamicMenuFactoryGenerator(menuEntries)
	tvg := tvgf(we.userState)

	web.Get("/wiki", we.GenerateWikiRedirect())
	web.Get("/edit/(.*)", wikiCP.WrapWebHandle(we.GenerateWikiEditForm(), tvg))
	web.Get("/wiki/(.*)", wikiCP.WrapWebHandle(we.GenerateShowWiki(), tvg))
	web.Post("/wiki", we.GenerateCreateOrUpdateWiki()) // Create or update pages
	web.Get("/css/wiki.css", we.GenerateCSS(wikiCP.ColorScheme))
}

func (we *WikiEngine) CreatePage(pageid string) string {
	if pageid == "edit" || pageid == "create" {
		return "Can not create a page named " + pageid
	}
	if pageid != CleanUpUserInput(pageid) {
		return "Can not create a page named " + pageid
	}
	for fieldName, defaultText := range wikiFields {
		err := we.wikiState.pages.Set(pageid, fieldName, defaultText)
		if err != nil {
			panic("ERROR: Can not create wiki page (" + fieldName + ")!")
		}
	}
	return "OK, created a page named " + pageid
}

func (we *WikiEngine) RemovePage(pageid string) {
	for fieldName, _ := range wikiFields {
		err := we.wikiState.pages.Del(pageid, fieldName)
		if err != nil {
			panic("ERROR: Can not remove wiki page (" + fieldName + ")!")
		}
	}
}

func (we *WikiEngine) ChangePage(pageid, newtitle, newtext string) {
	newtitle = CleanUpUserInput(newtitle)
	newtext = CleanUpUserInput(newtext)
	err := we.wikiState.pages.Set(pageid, "title", newtitle)
	if err != nil {
		panic("ERROR: Can not set wiki page title!")
	}
	err = we.wikiState.pages.Set(pageid, "text", newtext)
	if err != nil {
		panic("ERROR: Can not set wiki page text!")
	}
}

// Get a wiki page by page id, either raw or formatted
func (we *WikiEngine) GetText(pageid string, formatted bool) string {
	text, err := we.wikiState.pages.Get(pageid, "text")
	if err != nil {
		return "No text"
	}
	if formatted {

		// Wiki links
		re := regexp.MustCompile("\\[\\[(.*?)\\]\\]")
		text = re.ReplaceAllString(text, "<a href='/wiki/$1'>$1</a>")

		// Markdown
		text = string(blackfriday.MarkdownCommon([]byte(text)))
	}
	return text
}

func (we *WikiEngine) GetTitle(pageid string) string {
	retval, err := we.wikiState.pages.Get(pageid, "title")
	if err != nil {
		return "Untitled"
	}
	return retval
}

func (we *WikiEngine) HasPage(pageid string) bool {
	has, err := we.wikiState.pages.Exists(pageid)
	if err != nil {
		return false
	}
	return has
}

func (we *WikiEngine) GenerateCreateOrUpdateWiki() SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := GetBrowserUsername(ctx)
		if username == "" {
			return "No user logged in"
		}
		if !we.userState.IsLoggedIn(username) {
			return "Not logged in"
		}
		pageid := CleanUpUserInput(ctx.Params["id"])
		title := CleanUpUserInput(ctx.Params["title"])
		text := CleanUpUserInput(ctx.Params["text"])

		if !we.HasPage(pageid) {
			we.CreatePage(pageid)
		}
		we.ChangePage(pageid, title, text)

		//ctx.SetHeader("Refresh", "0; url=/wiki/" + pageid, true)
		return "/wiki/" + pageid
	}
}

func (we *WikiEngine) GenerateWikiEditForm() WebHandle {
	return func(ctx *web.Context, pageid string) string {
		username := GetBrowserUsername(ctx)
		if username == "" {
			return "No user logged in"
		}
		if !we.userState.IsLoggedIn(username) {
			return "Not logged in"
		}
		title := we.GetTitle(pageid)
		text := we.GetText(pageid, false)

		retval := "Page id: <input size='60' type='text' id='pageId' value='" + pageid + "'><br />"
		retval += "Page title: <input size='60' type='text' id='pageTitle' value='" + title + "'><br />"
		retval += "<textarea rows='20' cols='20' id='pageText'>" + text + "</textarea><br />"
		retval += JS("function save() { $.post('/wiki', {id:$('#pageId').val(), title:$('#pageTitle').val(), text:$('#pageText').val()}, function(data) { window.location.href=data; }); }")
		retval += "<button onClick='save();'>Save</button>"
		return retval
	}
}

func (we *WikiEngine) GenerateShowWiki() WebHandle {
	return func(ctx *web.Context, pageid string) string {
		retval := ""
		// Always show the wiki page
		// TODO: Add a feature for marking wiki pages as unofficial and/or locked?
		if we.HasPage(pageid) {
			retval += "<h1>" + we.GetTitle(pageid) + "</h1>"
			retval += we.GetText(pageid, true) + "<br />"
		} else {
			retval += "<h1>No such page: " + pageid + "</h1>"
		}
		// Display edit or create buttons if the user is logged in
		username := GetBrowserUsername(ctx)
		if (username != "") && we.userState.IsLoggedIn(username) {
			if we.HasPage(pageid) {
				retval += "<br /><button id='btnEdit'>Edit</button><br />"
				retval += JS(OnClick("#btnEdit", Redirect("/edit/"+pageid)))
			} else {
				retval += "<br /><button id='btnCreate'>Create</button><br />"
				retval += JS(OnClick("#btnCreate", Redirect("/edit/"+pageid)))
			}
		}
		return retval
	}
}

func (we *WikiEngine) GenerateWikiRedirect() SimpleContextHandle {
	return func(ctx *web.Context) string {
		ctx.SetHeader("Refresh", "0; url=/wiki/main", true)
		return ""
	}
}

func (we *WikiEngine) GenerateCSS(cs *ColorScheme) SimpleContextHandle {
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

#wikiText {
	background-color: white;
}

`
		//
	}
}
