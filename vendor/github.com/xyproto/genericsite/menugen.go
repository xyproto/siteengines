package genericsite

import (
	"github.com/xyproto/onthefly"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/webhandle"
	"net/http"
	"strconv"
)

type (
	MenuEntry struct {
		id   string
		text string
		url  string
	}
	MenuEntries []*MenuEntry
)

var menuIdCounter = 0

// Generate a new menu ID
func (me *MenuEntry) autoId() string {
	newId := ""

	// Start with the menu item number and increase the counter
	newId += strconv.Itoa(menuIdCounter)
	menuIdCounter++

	// Then add the first letter of the menu text
	if len(me.text) > 0 {
		newId += string(me.text[0])
	}

	return newId
}

// Takes something like "Admin:/admin" and returns a *MenuEntry
func NewMenuEntry(text_and_url string) *MenuEntry {
	var me MenuEntry
	me.text, me.url = webhandle.HostPortSplit(text_and_url)
	me.id = me.autoId()
	return &me
}

func Links2menuEntries(links []string) MenuEntries {
	menuEntries := make(MenuEntries, len(links))
	for i, text_and_url := range links {
		menuEntries[i] = NewMenuEntry(text_and_url)
	}
	return menuEntries
}

// Generate tags for the menu based on a list of "MenuDescription:/menu/url"
func MenuSnippet(menuEntries MenuEntries) *onthefly.Page {
	var a, li, sep *onthefly.Tag

	page, ul := onthefly.StandaloneTag("ul")
	ul.AddAttrib("class", "menuList")
	//ul.AddStyle("list-style-type", "none")
	//ul.AddStyle("float", "left")
	//ul.AddStyle("margin", "0")

	for i, menuEntry := range menuEntries {

		li = ul.AddNewTag("li")
		li.AddAttrib("class", "menuEntry")

		// TODO: Make sure not duplicate ids are added for two menu entries named "Hi there" and "Hi you". Add i to string?
		menuId := "menu" + menuEntry.id
		li.AddAttrib("id", menuId)

		// All menu entries are now hidden by default!
		//li.AddStyle("display", "none")
		//li.AddStyle("display", "inline")

		li.SansSerif()
		//li.CustomSansSerif("Armata")

		// For every element, except the first one
		if i > 0 {
			// Insert a '|' character in a div
			sep = li.AddNewTag("div")
			sep.AddContent("|")
			sep.AddAttrib("class", "separator")
		}

		a = li.AddNewTag("a")
		a.AddAttrib("class", "menulink")
		a.AddAttrib("href", menuEntry.url)
		a.AddContent(menuEntry.text)

	}

	return page
}

// Checks if a *MenuEntry exists in a []*MenuEntry (MenuEntries)
func HasEntry(checkEntry *MenuEntry, menuEntries MenuEntries) bool {
	for _, menuEntry := range menuEntries {
		if menuEntry.url == checkEntry.url {
			return true
		}
	}
	return false
}

func AddIfNotAdded(url string, filteredMenuEntries *MenuEntries, menuEntry *MenuEntry) {
	//if currentMenuURL != url {
	if menuEntry.url == url {
		if !HasEntry(menuEntry, *filteredMenuEntries) {
			*filteredMenuEntries = append(*filteredMenuEntries, menuEntry)
		}
	}
	//}
}

/*
 * Functions that generate functions that generate content that can be used in templates.
 * type TemplateValues map[string]string
 * type TemplateValueGenerator func(*web.Context) TemplateValues
 * type TemplateValueGeneratorFactory func(*UserState) TemplateValueGenerator
 */
// TODO: Take the same parameters as the old menu generating code
// TODO: Put one if these in each engine then combine them somehow
// TODO: Check for the menyEntry.url first, then check the rights, not the other way around
// TODO: Fix and refactor this one
// TODO: Check the user status _once_, and the admin status _once_, then generate the menu
// TODO: Some way of marking menu entries as user, admin or other rights. Add a group system?
func DynamicMenuFactoryGenerator(menuEntries MenuEntries) TemplateValueGeneratorFactory {
	return func(state pinterface.IUserState) webhandle.TemplateValueGenerator {
		return func(w http.ResponseWriter, req *http.Request) onthefly.TemplateValues {

			userRights := state.UserRights(req)
			adminRights := state.AdminRights(req)

			var filteredMenuEntries MenuEntries
			var logoutEntry *MenuEntry = nil

			// Build up filteredMenuEntries based on what should be shown or not
			for _, menuEntry := range menuEntries {

				// Don't add duplicates
				if HasEntry(menuEntry, filteredMenuEntries) {
					continue
				}

				// Add this one last
				if menuEntry.url == "/logout" {
					if userRights {
						logoutEntry = menuEntry
					}
					continue
				}

				// Always show the Overview menu
				AddIfNotAdded("/", &filteredMenuEntries, menuEntry)
				//if menuEntry.url == "/" {
				//	if !HasEntry(menuEntry, filteredMenuEntries) {
				//		filteredMenuEntries = append(filteredMenuEntries, menuEntry)
				//	}
				//}

				// If logged in, show Logout and the content
				if userRights {

					// Add every link except the current page we're on
					//if menuEntry.url != currentMenuURL {
					if !HasEntry(menuEntry, filteredMenuEntries) {
						if (menuEntry.url != "/login") && (menuEntry.url != "/register") && (menuEntry.url != "/admin") {
							filteredMenuEntries = append(filteredMenuEntries, menuEntry)
						}
					}
					//}

					// Show admin content
					if adminRights {
						AddIfNotAdded("/admin", &filteredMenuEntries, menuEntry)
					}
				} else {
					// Only show Login and Register
					AddIfNotAdded("/login", &filteredMenuEntries, menuEntry)
					AddIfNotAdded("/register", &filteredMenuEntries, menuEntry)
				}

			}

			if logoutEntry != nil {
				AddIfNotAdded("/logout", &filteredMenuEntries, logoutEntry)
			}

			page := MenuSnippet(filteredMenuEntries)
			retval := page.String()

			// TODO: Return the CSS as well somehow
			//css := page.CSS()

			return onthefly.TemplateValues{"menu": retval}
		}
	}
}

// Combines two TemplateValueGenerators into one TemplateValueGenerator by adding the strings per key
func TemplateValueGeneratorCombinator(tvg1, tvg2 webhandle.TemplateValueGenerator) webhandle.TemplateValueGenerator {
	return func(w http.ResponseWriter, req *http.Request) onthefly.TemplateValues {
		tv1 := tvg1(w, req)
		tv2 := tvg2(w, req)
		for key, value := range tv2 {
			// TODO: Check if key exists in tv1 first
			tv1[key] += value
		}
		return tv1
	}
}
