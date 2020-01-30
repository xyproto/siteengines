package genericsite

import (
	"fmt"
	"net/http"
	"time"

	"github.com/drbawb/mustache"
	"github.com/gorilla/mux"
	"github.com/xyproto/onthefly"
	"github.com/xyproto/permissions2"
	"github.com/xyproto/pinterface"
	"github.com/xyproto/webhandle"
)

type (

	// The main structure, defining the look and feel of the page
	ContentPage struct {
		GeneratedCSSurl          string
		ExtraCSSurls             []string
		JqueryJSurl              string
		Faviconurl               string
		BgImageURL               string
		StretchBackground        bool
		Title                    string
		Subtitle                 string
		ContentTitle             string
		ContentHTML              string
		HeaderJS                 string
		ContentJS                string
		SearchButtonText         string
		SearchURL                string
		FooterText               string
		BackgroundTextureURL     string
		DarkBackgroundTextureURL string
		FooterTextColor          string
		FooterColor              string
		UserState                pinterface.IUserState
		RoundedLook              bool
		Url                      string
		ColorScheme              *ColorScheme
		SearchBox                bool
		GoogleFonts              []string
		CustomSansSerif          string
		CustomSerif              string
	}

	// Content page generator
	CPgen (func(userState permissions.UserState) *ContentPage)

	// Collection of ContentPages
	PageCollection []ContentPage

	// Every input from the user must be intitially stored in a UserInput variable, not in a string!
	// This is just to be aware of which data one should be careful with, and to keep it clean.
	UserInput string

	ColorScheme struct {
		Darkgray           string
		Nicecolor          string
		Menu_link          string
		Menu_hover         string
		Menu_active        string
		Default_background string
		TitleText          string
	}

	// Base content page
	BaseCP func(state pinterface.IUserState) *ContentPage

	TemplateValueGeneratorFactory func(pinterface.IUserState) webhandle.TemplateValueGenerator
)

// The default settings
// Do not publish this page directly, but use it as a basis for the other pages
func DefaultCP(userState pinterface.IUserState) *ContentPage {
	var cp ContentPage
	cp.GeneratedCSSurl = "/css/style.css"
	cp.ExtraCSSurls = []string{"/css/menu.css"}
	// TODO: fallback to local jquery.min.js, google how
	cp.JqueryJSurl = "//ajax.googleapis.com/ajax/libs/jquery/2.0.0/jquery.min.js" // "/js/jquery-2.0.0.js"
	cp.Faviconurl = "/img/favicon.ico"
	cp.ContentTitle = "NOP"
	cp.ContentHTML = "NOP NOP NOP"
	cp.ContentJS = ""
	cp.HeaderJS = ""
	cp.SearchButtonText = "Search"
	cp.SearchURL = "/search"
	cp.SearchBox = true

	// http://wptheming.wpengine.netdna-cdn.com/wp-content/uploads/2010/04/gray-texture.jpg
	// TODO: Draw these two backgroundimages with a canvas instead
	cp.BackgroundTextureURL = "" // "/img/gray.jpg"
	// http://turbo.designwoop.com/uploads/2012/03/16_free_subtle_textures_subtle_dark_vertical.jpg
	cp.DarkBackgroundTextureURL = "/img/darkgray.jpg"

	cp.FooterColor = "black"
	cp.FooterTextColor = "#303040"

	cp.FooterText = "NOP"

	cp.UserState = userState
	cp.RoundedLook = false

	cp.Url = "/" // To be filled in when published

	// The default color scheme
	var cs ColorScheme
	cs.Darkgray = "#202020"
	cs.Nicecolor = "#5080D0"   // nice blue
	cs.Menu_link = "#c0c0c0"   // light gray
	cs.Menu_hover = "#efefe0"  // light gray, somewhat yellow
	cs.Menu_active = "#ffffff" // white
	cs.Default_background = "#000030"
	cs.TitleText = "#303030"

	cp.ColorScheme = &cs

	cp.GoogleFonts = []string{"Armata", "IM Fell English SC"}
	cp.CustomSansSerif = "" // Use the default sans serif
	cp.CustomSerif = "IM Fell English SC"

	return &cp
}

func genericPageBuilder(cp *ContentPage) *onthefly.Page {
	// TODO: Record the time from one step out, because content may be generated and inserted into this generated conten
	startTime := time.Now()

	page := onthefly.NewHTML5Page(cp.Title + " " + cp.Subtitle)

	page.LinkToCSS(cp.GeneratedCSSurl)
	for _, cssurl := range cp.ExtraCSSurls {
		page.LinkToCSS(cssurl)
	}
	page.LinkToJS(cp.JqueryJSurl)
	page.LinkToFavicon(cp.Faviconurl)

	onthefly.AddHeader(page, cp.HeaderJS)
	onthefly.AddGoogleFonts(page, cp.GoogleFonts)
	onthefly.AddBodyStyle(page, cp.BgImageURL, cp.StretchBackground)
	AddTopBox(page, cp.Title, cp.Subtitle, cp.SearchURL, cp.SearchButtonText, cp.BackgroundTextureURL, cp.RoundedLook, cp.ColorScheme, cp.SearchBox)

	// TODO: Move the menubox into the TopBox

	AddMenuBox(page, cp.DarkBackgroundTextureURL, cp.CustomSansSerif)

	AddContent(page, cp.ContentTitle, cp.ContentHTML+onthefly.DocumentReadyJS(cp.ContentJS))

	elapsed := time.Since(startTime)
	AddFooter(page, cp.FooterText, cp.FooterTextColor, cp.FooterColor, elapsed)

	return page
}

// Publish a list of ContentPages, a colorscheme and template content
func PublishCPs(r *mux.Router, userState pinterface.IUserState, pc PageCollection, cs *ColorScheme, tvgf TemplateValueGeneratorFactory, cssurl string) {
	// For each content page in the page collection
	for _, cp := range pc {
		// TODO: different css urls for all of these?
		cp.Pub(r, userState, cp.Url, cssurl, cs, tvgf(userState))
	}
}

// Some Engines like Admin must be served separately
// jquerypath is ie "/js/jquery.2.0.0.js", will then serve the file at static/js/jquery.2.0.0.js
func ServeSite(r *mux.Router, basecp BaseCP, userState pinterface.IUserState, cps PageCollection, tvgf TemplateValueGeneratorFactory, jquerypath string) {
	cs := basecp(userState).ColorScheme
	PublishCPs(r, userState, cps, cs, tvgf, "/css/menu.css")

	// TODO: Add fallback to this local version
	webhandle.Publish(r, jquerypath, "static"+jquerypath)

	// TODO: Generate these
	webhandle.Publish(r, "/robots.txt", "static/various/robots.txt")
	webhandle.Publish(r, "/sitemap_index.xml", "static/various/sitemap_index.xml")
}

// Create a web.go compatible function that returns a string that is the HTML for this page
func GenerateHTMLwithTemplate(page *onthefly.Page, tvg webhandle.TemplateValueGenerator) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		values := tvg(w, req)
		fmt.Fprintf(w, "%s", mustache.Render(page.GetXML(true), values))
	}
}

// CSS for the menu, and a bit more
func GenerateMenuCSS(state pinterface.IUserState, stretchBackground bool, cs *ColorScheme) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/css")

		// one of the extra css files that are loaded after the main style
		retval := mustache.Render(menustyle_tmpl, cs)

		// The load order of background-color, background-size and background-image
		// is actually significant in some browsers! Do not reorder lightly.
		if stretchBackground {
			retval = "body {\nbackground-color: " + cs.Default_background + ";\nbackground-size: cover;\n}\n" + retval
		} else {
			retval = "body {\nbackground-color: " + cs.Default_background + ";\n}\n" + retval
		}
		retval += ".titletext { display: inline; }"

		fmt.Fprintf(w, "%s", retval)
	}
}

// Make an html and css page available
func (cp *ContentPage) Pub(r *mux.Router, userState pinterface.IUserState, url, cssurl string, cs *ColorScheme, tvg webhandle.TemplateValueGenerator) {
	genericpage := genericPageBuilder(cp)
	r.HandleFunc(url, GenerateHTMLwithTemplate(genericpage, tvg))
	r.HandleFunc(cp.GeneratedCSSurl, webhandle.GenerateCSS(genericpage))
	r.HandleFunc(cssurl, GenerateMenuCSS(userState, cp.StretchBackground, cs))
}

// TODO: Write a function for rendering a StandaloneTag inside a Page by the use of template {{{placeholders}}}

// Render a page by inserting data at the {{{placeholders}}} for both html and css
func RenderPage(page *onthefly.Page, templateContents map[string]string) (string, string) {
	// Note that the whitespace formatting of the generated html matter for the menu layout!
	return mustache.Render(page.String(), templateContents), mustache.Render(page.GetCSS(), templateContents)
}

// Wrap a lonely string in an entire webpage
func (cp *ContentPage) Surround(s string, templateContents map[string]string) (string, string) {
	cp.ContentHTML = s
	page := genericPageBuilder(cp)
	return RenderPage(page, templateContents)
}

// Uses a given WebHandle as the contents for the the ContentPage contents
func (cp *ContentPage) WrapWebHandle(r *mux.Router, wh func(string) string, tvg webhandle.TemplateValueGenerator) func(string, http.ResponseWriter, *http.Request) {
	return func(val string, w http.ResponseWriter, req *http.Request) {
		html, css := cp.Surround(wh(val), tvg(w, req))
		r.HandleFunc(cp.GeneratedCSSurl, func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("Content-Type", "text/css")
			fmt.Fprintf(w, "%s", css)
		})
		fmt.Fprintf(w, "%s", html)
	}
}

// Uses a given SimpleContextHandle as the contents for the the ContentPage contents
func (cp *ContentPage) WrapSimpleContextHandle(r *mux.Router, sch func(w http.ResponseWriter, req *http.Request) string, tvg webhandle.TemplateValueGenerator) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		html, css := cp.Surround(sch(w, req), tvg(w, req))
		r.HandleFunc(cp.GeneratedCSSurl, func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("Content-Type", "text/css")
			fmt.Fprintf(w, "%s", css)
		})
		fmt.Fprintf(w, "%s", html)
	}
}

// Template for the CSS for the menu
const menustyle_tmpl = `
a {
  text-decoration: none;
  color: #303030;
  font-weight: regular;
}

a:link {
  color: {{Menu_link}};
}

a:visited {
  color: {{Menu_link}};
}

a:hover {
  color: {{Menu_hover}};
}

a:active {
  color: {{Menu_active}};
}

.menuEntry {
  display: inline;
}

.menuList {
  list-style-type: none;
  float: left;
  margin: 0;
}

.separator {
  display: inline;
  color: #a0a0a0;
}
`
