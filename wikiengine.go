package siteengines

import (
	"regexp"
	"strings"

	"github.com/hoisie/web"
	"github.com/russross/blackfriday"
	. "github.com/xyproto/genericsite"
	. "github.com/xyproto/onthefly"
	"github.com/xyproto/pinterface"
	. "github.com/xyproto/webhandle"
)

// An Engine is a specific piewe.of a website
// This part handles the "wiki" pages

// TODO: Create a page that lists all the wiki pages

// TODO: Add the wiki pages to the search engine somehow (and the other engines too, like the chat)

type WikiEngine struct {
	state     pinterface.IUserState
	wikiState *WikiState
}

type WikiState struct {
	pages pinterface.IHashMap // All the pages
}

var (
	wikiFields = map[string]string{
		"title": "Untitled",
		"text":  "No text",
	}
)

func NewWikiEngine(userState pinterface.IUserState) (*WikiEngine, error) {
	creator := userState.Creator()

	wikiState := new(WikiState)
	if pagesHashMap, err := creator.NewHashMap("pages"); err != nil {
		return nil, err
	} else {
		wikiState.pages = pagesHashMap
	}

	return &WikiEngine{userState, wikiState}, nil
}

func (we *WikiEngine) ServePages(basecp BaseCP, menuEntries MenuEntries) {
	wikiCP := basecp(we.state)
	wikiCP.ContentTitle = "Wiki"
	wikiCP.ExtraCSSurls = append(wikiCP.ExtraCSSurls, "/css/wiki.css")

	tvgf := DynamicMenuFactoryGenerator(menuEntries)
	tvg := tvgf(we.state)

	web.Get("/wiki", we.GenerateWikiRedirect())                                         // Redirect to /wiki/main
	web.Get("/wikiedit/(.*)", wikiCP.WrapWebHandle(we.GenerateWikiEditForm(), tvg))     // Form for editing wiki pages
	web.Get("/wikisource/(.*)", wikiCP.WrapWebHandle(we.GenerateWikiViewSource(), tvg)) // Page for viewing the source
	web.Get("/wikidelete/(.*)", wikiCP.WrapWebHandle(we.GenerateWikiDeleteForm(), tvg)) // Form for deleting wiki pages
	web.Get("/wiki/(.*)", wikiCP.WrapWebHandle(we.GenerateShowWiki(), tvg))             // Displaying wiki pages
	web.Get("/wikipages", wikiCP.WrapSimpleContextHandle(we.GenerateListPages(), tvg))  // Listing wiki pages
	web.Post("/wiki", we.GenerateCreateOrUpdateWiki())                                  // Create or update pages
	web.Post("/wikideletenow", we.GenerateDeleteWikiNow())                              // Delete pages (admin only)
	web.Get("/css/wiki.css", we.GenerateCSS(wikiCP.ColorScheme))                        // CSS that is specific for wiki pages
}

func (we *WikiEngine) ListPages() string {
	pageids, err := we.wikiState.pages.GetAll()
	if err != nil {
		return ""
	}
	retval := ""
	for _, pageid := range pageids {
		retval += "<a href='/wiki/" + pageid + "'>" + pageid + "</a><br />"
	}
	return retval
}

func (we *WikiEngine) CreatePage(pageid string) string {
	if pageid != CleanUserInput(pageid) {
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

func (we *WikiEngine) DeletePage(pageid string) {
	err := we.wikiState.pages.Del(pageid)
	if err != nil {
		panic("ERROR: Can not remove wiki page (" + pageid + ")!")
	}
}

func (we *WikiEngine) ChangePage(pageid, newtitle, newtext string) {
	newtitle = CleanUserInput(newtitle)
	newtext = CleanUserInput(newtext)
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
		// Suggest "hi" as the default text
		// TODO: Use a poem generator instead
		return "hi"
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
		// Suggest a capitalized version of the page id as the default title
		return strings.Title(pageid)
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

func (we *WikiEngine) GenerateListPages() SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := we.state.Username(ctx.Request)
		if username == "" {
			return "No user logged in"
		}
		if !we.state.IsLoggedIn(username) {
			return "Not logged in"
		}
		retval := ""
		retval += "<h2>All wiki pages</h2>"
		retval += we.ListPages()
		retval += "<br />"
		retval += BackButton()
		return retval
	}
}

func (we *WikiEngine) GenerateCreateOrUpdateWiki() SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := we.state.Username(ctx.Request)
		if username == "" {
			return "No user logged in"
		}
		if !we.state.IsLoggedIn(username) {
			return "Not logged in"
		}
		pageid := CleanUserInput(ctx.Params["id"])
		title := CleanUserInput(ctx.Params["title"])
		text := CleanUserInput(ctx.Params["text"])

		if !we.HasPage(pageid) {
			we.CreatePage(pageid)
		}
		we.ChangePage(pageid, title, text)

		return "/wiki/" + pageid
	}
}

func (we *WikiEngine) GenerateDeleteWikiNow() SimpleContextHandle {
	return func(ctx *web.Context) string {
		username := we.state.Username(ctx.Request)
		if username == "" {
			return "No user logged in"
		}
		if !we.state.IsLoggedIn(username) {
			return "Not logged in"
		}
		if !we.state.IsAdmin(username) {
			return "Not admin"
		}
		pageid := CleanUserInput(ctx.Params["id"])

		if pageid == "" {
			return "Could not delete empty pageid"
		}

		if !we.HasPage(pageid) {
			return "Could not delete this wiki page: " + pageid
		}
		we.DeletePage(pageid)

		return "OK, page deleted: " + pageid

	}
}

func (we *WikiEngine) GenerateWikiEditForm() WebHandle {
	return func(ctx *web.Context, pageid string) string {
		username := we.state.Username(ctx.Request)
		if username == "" {
			return "No user logged in"
		}
		if !we.state.IsLoggedIn(username) {
			return "Not logged in"
		}

		pageid = CleanUserInput(pageid)
		title := we.GetTitle(pageid)
		text := we.GetText(pageid, false)

		retval := ""
		retval += "<h2>Create or edit</h2>"
		retval += "Page id: <input size='30' type='text' id='pageId' value='" + pageid + "'><br />"
		retval += "Page title: <input size='40' type='text' id='pageTitle' value='" + title + "'><br /><br />"
		retval += "<textarea rows='25' cols='120' id='pageText'>" + text + "</textarea><br /><br />"
		retval += JS("function save() { $.post('/wiki', {id:$('#pageId').val(), title:$('#pageTitle').val(), text:$('#pageText').val()}, function(data) { window.location.href=data; }); }")
		retval += "<button onClick='save();'>Save</button>"
		retval += BackButton()
		// Focus on the text
		retval += JS(Focus("#pageText") + "$('#pageText').select();")
		return retval
	}
}

func (we *WikiEngine) GenerateWikiViewSource() WebHandle {
	return func(ctx *web.Context, pageid string) string {
		username := we.state.Username(ctx.Request)
		if username == "" {
			return "No user logged in"
		}
		if !we.state.IsLoggedIn(username) {
			return "Not logged in"
		}

		pageid = CleanUserInput(pageid)
		title := we.GetTitle(pageid)
		text := we.GetText(pageid, false)

		retval := ""
		retval += "<h2>View source</h2>"
		retval += "Page id: <input style='background-color: #e0e0e0;' readonly='readonly' size='30' type='text' id='pageId' value='" + pageid + "'><br />"
		retval += "Page title: <input style='background-color: #e0e0e0;' readonly='readonly' size='40' type='text' id='pageTitle' value='" + title + "'><br /><br />"
		retval += "<textarea style='background-color: #e0e0e0;' readonly='readonly' rows='25' cols='120' id='pageText'>" + text + "</textarea><br /><br />"
		retval += BackButton()
		return retval
	}
}

func (we *WikiEngine) GenerateWikiDeleteForm() WebHandle {
	return func(ctx *web.Context, pageid string) string {
		username := we.state.Username(ctx.Request)
		if username == "" {
			return "No user logged in"
		}
		if !we.state.IsLoggedIn(username) {
			return "Not logged in"
		}
		if !we.state.IsAdmin(username) {
			return "Must be admin"
		}

		pageid = CleanUserInput(pageid)

		retval := "<br />"
		retval += "Really delete " + pageid + "?<br />"
		retval += JS("function deletePage() { $.post('/wikideletenow', {id:'" + pageid + "'}, function(data) { $('#status').html(data) }); }")
		retval += "<button onClick='deletePage();'>Yes</button><br />"
		retval += "<label id='status'></label><br />"
		retval += BackButton()
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
		username := we.state.Username(ctx.Request)
		if (username != "") && we.state.IsLoggedIn(username) {
			if we.HasPage(pageid) {
				// Page actions for regular users for pages that exists and are not the main page
				if pageid != "main" {
					retval += "<br /><button id='btnEdit'>Edit</button>"
					retval += JS(OnClick("#btnEdit", Redirect("/wikiedit/"+pageid)))
					retval += "<button id='btnDelete'>Delete</button>"
					retval += JS(OnClick("#btnDelete", Redirect("/wikidelete/"+pageid)))
				} else {
					// Page actions for administrators on the main page
					if we.state.IsAdmin(username) {
						retval += "<br /><button id='btnEdit'>Edit</button>"
						retval += JS(OnClick("#btnEdit", Redirect("/wikiedit/"+pageid)))
					}
				}
				// Page actions for regular users for every page
				retval += "<button id='btnViewSource'>View source</button>"
				retval += JS(OnClick("#btnViewSource", Redirect("/wikisource/"+pageid)))
			} else {
				// Page actions for regular users for pages that does not exist yet
				retval += "<br /><button id='btnCreate'>Create</button>"
				retval += JS(OnClick("#btnCreate", Redirect("/wikiedit/"+pageid)))
			}
		}
		retval += BackButton()
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
