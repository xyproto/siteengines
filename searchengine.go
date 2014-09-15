package siteengines

import (
	"net/http"
	"strings"
	"time"

	. "github.com/xyproto/genericsite"
	. "github.com/xyproto/onthefly"
	"github.com/xyproto/permissions"
	. "github.com/xyproto/webhandle"
)

const (
	FOUND_IN_URL = iota
	FOUND_IN_TITLE
	FOUND_IN_TEXT
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Search a list of ContentPage for a given searchText
// Returns a list of urls or an empty list, a list of page titles and the string that was actually searched for
func searchResults(userSearchText UserInput, pc PageCollection) ([]string, []string, string, []int) {
	// Search for maximum 100 letters, lowercase and trimmed
	searchText := strings.ToLower(strings.TrimSpace(string(userSearchText)[:min(100, len(string(userSearchText)))]))

	if searchText == "" {
		// No search results for the empty string
		return []string{}, []string{}, "", []int{}
	}

	var matches, titles []string
	var foundWhere []int
	// TODO: Refactor to get less repetition
	for _, cp := range pc {
		if strings.Contains(strings.ToLower(cp.ContentTitle), searchText) {
			// Check if the url is already in the matches slices
			found := false
			for _, url := range matches {
				if url == cp.Url {
					found = true
					break
				}
			}
			// If not, add it
			if !found {
				matches = append(matches, cp.Url)
				titles = append(titles, cp.ContentTitle)
				foundWhere = append(foundWhere, FOUND_IN_TITLE)
				continue
			}
		}
		if strings.Contains(strings.ToLower(cp.Url), searchText) {
			// Check if the url is already in the matches slices
			found := false
			for _, url := range matches {
				if url == cp.Url {
					found = true
					break
				}
			}
			// If not, add it
			if !found {
				matches = append(matches, cp.Url)
				titles = append(titles, cp.ContentTitle)
				foundWhere = append(foundWhere, FOUND_IN_URL)
				continue
			}
		}
		if strings.Contains(strings.ToLower(cp.ContentHTML), searchText) {
			// Check if the url is already in the matches slices
			found := false
			for _, url := range matches {
				if url == cp.Url {
					found = true
					break
				}
			}
			// If not, add it
			if !found {
				matches = append(matches, cp.Url)
				titles = append(titles, cp.ContentTitle)
				foundWhere = append(foundWhere, FOUND_IN_TEXT)
				continue
			}
		}
	}
	return matches, titles, searchText, foundWhere
}

// Generate a search handle. This is done in order to be able to modify the cp
// Searches a list of ContentPage structs
func GenerateSearchHandle(pc PageCollection) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		q := GetParam(req, "q")
		if q != "" {
			searchText := UserInput(q)
			content := "Search: " + string(searchText)
			nl := TagString("br")
			content += nl + nl
			startTime := time.Now()
			urls, titles, searchedFor, foundWhere := searchResults(searchText, pc)
			elapsed := time.Since(startTime)
			page, p := StandaloneTag("p")
			if len(urls) == 0 {
				p.AddContent("No results found")
				p.AddNewTag("br")
			} else {
				for _, foundType := range []int{FOUND_IN_URL, FOUND_IN_TITLE, FOUND_IN_TEXT} {
					for i, url := range urls {
						// Add url-matches first, then title-matches then text-matches
						if foundWhere[i] == foundType {
							a := p.AddNewTag("a")
							a.AddAttrib("id", "searchresult")
							a.AddStyle("color", "red")
							a.AddAttrib("href", url)
							a.AddContent(titles[i])
							font := p.AddNewTag("font")
							if foundType == FOUND_IN_URL {
								font.AddContent(" - url contains \"" + searchedFor + "\"")
							} else if foundType == FOUND_IN_TITLE {
								font.AddContent(" - title contains \"" + searchedFor + "\"")
							} else {
								font.AddContent(" - page contains \"" + searchedFor + "\"")
							}
							p.AddNewTag("br")
						}
					}
				}
			}
			p.AddNewTag("br")
			p.AddLastContent("Search took: " + elapsed.String())
			Ret(w, page.GetHTML()) //GenerateHTMLwithTemplate(page, Kake())
			return
		}
		Ret(w, "Invalid parameters")
	}
}

func ServeSearchPages(mux *http.ServeMux, basecp BaseCP, state *permissions.UserState, cps PageCollection, cs *ColorScheme, tpg TemplateValueGenerator) {
	searchCP := basecp(state)
	searchCP.ContentTitle = "Search results"
	searchCP.ExtraCSSurls = append(searchCP.ExtraCSSurls, "/css/search.css")

	// Note, no slash between "search" and "(.*)". A typical search is "/search?q=blabla"
	mux.HandleFunc("/search/", searchCP.WrapHandle(mux, GenerateSearchHandle(cps), tpg))
	mux.HandleFunc("/css/search.css", GenerateSearchCSS(cs))
}

func GenerateSearchCSS(cs *ColorScheme) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/css")
		Ret(w, `
#searchresult {
	color: `+cs.Nicecolor+`;
	text-decoration: underline;
}
`)
		//
	}
}
