package siteengines

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/russross/blackfriday"
	. "github.com/xyproto/genericsite"
	. "github.com/xyproto/onthefly"
	"github.com/xyproto/permissions"
	"github.com/xyproto/simpleredis"
	. "github.com/xyproto/webhandle"
)

// An Engine is a specific piewe.of a website
// This part handles the "wiki" pages

// TODO: Create a page that lists all the wiki pages

// TODO: Add the wiki pages to the search engine somehow (and the other engines too, like the chat)

type WikiEngine struct {
	state     *permissions.UserState
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

func NewWikiEngine(state *permissions.UserState) *WikiEngine {
	pool := state.GetPool()

	wikiState := new(WikiState)
	wikiState.pages = simpleredis.NewHashMap(pool, "pages")
	wikiState.pages.SelectDatabase(state.GetDatabaseIndex())
	wikiState.pool = pool

	return &WikiEngine{state, wikiState}
}

func (we *WikiEngine) ServePages(mux *http.ServeMux, basecp BaseCP, menuEntries MenuEntries) {
	wikiCP := basecp(we.state)
	wikiCP.ContentTitle = "Wiki"
	wikiCP.ExtraCSSurls = append(wikiCP.ExtraCSSurls, "/css/wiki.css")

	tvgf := DynamicMenuFactoryGenerator(menuEntries)
	tvg := tvgf(we.state)

	mux.HandleFunc("/wiki", we.GenerateWikiRedirect())                                           // Redirect to /wiki/main
	mux.HandleFunc("/wikiedit/(.*)", wikiCP.WrapHandle(mux, we.GenerateWikiEditForm(), tvg))     // Form for editing wiki pages
	mux.HandleFunc("/wikisource/(.*)", wikiCP.WrapHandle(mux, we.GenerateWikiViewSource(), tvg)) // Page for viewing the source
	mux.HandleFunc("/wikidelete/(.*)", wikiCP.WrapHandle(mux, we.GenerateWikiDeleteForm(), tvg)) // Form for deleting wiki pages
	mux.HandleFunc("/wiki/(.*)", wikiCP.WrapHandle(mux, we.GenerateShowWiki(), tvg))             // Displaying wiki pages
	mux.HandleFunc("/wikipages", wikiCP.WrapHandle(mux, we.GenerateListPages(), tvg))            // Listing wiki pages
	mux.HandleFunc("/wiki", we.GenerateCreateOrUpdateWiki())                                     // Create or update pages
	mux.HandleFunc("/wikideletenow", we.GenerateDeleteWikiNow())                                 // Delete pages (admin only)
	mux.HandleFunc("/css/wiki.css", we.GenerateCSS(wikiCP.ColorScheme))                          // CSS that is specific for wiki pages
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

func (we *WikiEngine) GenerateListPages() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		username := we.state.GetUsername(req)
		if username == "" {
			Ret(w, "No user logged in")
			return
		}
		if !we.state.IsLoggedIn(username) {
			Ret(w, "Not logged in")
			return
		}
		retval := ""
		retval += "<h2>All wiki pages</h2>"
		retval += we.ListPages()
		retval += "<br />"
		retval += BackButton()
		Ret(w, retval)
	}
}

func (we *WikiEngine) GenerateCreateOrUpdateWiki() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		username := we.state.GetUsername(req)
		if username == "" {
			Ret(w, "No user logged in")
		}
		if !we.state.IsLoggedIn(username) {
			Ret(w, "Not logged in")
		}
		pageid := CleanUserInput(GetParam(req, "id"))
		title := CleanUserInput(GetParam(req, "title"))
		text := CleanUserInput(GetParam(req, "text"))

		if !we.HasPage(pageid) {
			we.CreatePage(pageid)
		}
		we.ChangePage(pageid, title, text)

		Ret(w, "/wiki/"+pageid)
	}
}

func (we *WikiEngine) GenerateDeleteWikiNow() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		username := we.state.GetUsername(req)
		if username == "" {
			Ret(w, "No user logged in")
			return
		}
		if !we.state.IsLoggedIn(username) {
			Ret(w, "Not logged in")
			return
		}
		if !we.state.IsAdmin(username) {
			Ret(w, "Not admin")
			return
		}
		pageid := CleanUserInput(GetParam(req, "id"))

		if pageid == "" {
			Ret(w, "Could not delete empty pageid")
			return
		}

		if !we.HasPage(pageid) {
			Ret(w, "Could not delete this wiki page: "+pageid)
			return
		}
		we.DeletePage(pageid)

		Ret(w, "OK, page deleted: "+pageid)
	}
}

func (we *WikiEngine) GenerateWikiEditForm() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		pageid := GetLast(req.URL)
		username := we.state.GetUsername(req)
		if username == "" {
			Ret(w, "No user logged in")
			return
		}
		if !we.state.IsLoggedIn(username) {
			Ret(w, "Not logged in")
			return
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
		Ret(w, retval)
	}
}

func (we *WikiEngine) GenerateWikiViewSource() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		pageid := GetLast(req.URL)
		username := we.state.GetUsername(req)
		if username == "" {
			Ret(w, "No user logged in")
			return
		}
		if !we.state.IsLoggedIn(username) {
			Ret(w, "Not logged in")
			return
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
		Ret(w, retval)
	}
}

func (we *WikiEngine) GenerateWikiDeleteForm() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		pageid := GetLast(req.URL)
		username := we.state.GetUsername(req)
		if username == "" {
			Ret(w, "No user logged in")
			return
		}
		if !we.state.IsLoggedIn(username) {
			Ret(w, "Not logged in")
			return
		}
		if !we.state.IsAdmin(username) {
			Ret(w, "Must be admin")
			return
		}

		pageid = CleanUserInput(pageid)

		retval := "<br />"
		retval += "Really delete " + pageid + "?<br />"
		retval += JS("function deletePage() { $.post('/wikideletenow', {id:'" + pageid + "'}, function(data) { $('#status').html(data) }); }")
		retval += "<button onClick='deletePage();'>Yes</button><br />"
		retval += "<label id='status'></label><br />"
		retval += BackButton()
		Ret(w, retval)
	}
}

func (we *WikiEngine) GenerateShowWiki() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		pageid := GetLast(req.URL)
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
		username := we.state.GetUsername(req)
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
		Ret(w, retval)
	}
}

func (we *WikiEngine) GenerateWikiRedirect() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Refresh", "0; url=/wiki/main")
	}
}

func (we *WikiEngine) GenerateCSS(cs *ColorScheme) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/css")
		Ret(w, `
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

`)
		//
	}
}
