package genericsite

// Various elements of a webpage

import (
	"strings"
	"time"

	"github.com/xyproto/onthefly"
)

func AddTopBox(page *onthefly.Page, title, subtitle, searchURL, searchButtonText, backgroundTextureURL string, roundedLook bool, cs *ColorScheme, addSearchBox bool) (*onthefly.Tag, error) {
	body, err := page.GetTag("body")
	if err != nil {
		return nil, err
	}

	div := body.AddNewTag("div")
	div.AddAttrib("id", "topbox")
	div.AddStyle("display", "block")
	div.AddStyle("width", "100%")
	div.AddStyle("margin", "0")
	div.AddStyle("padding", "0 0 1em 0")
	div.AddStyle("top", "0")
	div.AddStyle("left", "0")
	div.AddStyle("background-color", cs.Darkgray)
	div.AddStyle("position", "fixed")
	div.AddStyle("display", "block")

	titlebox := AddTitleBox(div, title, subtitle, cs)
	titlebox.AddAttrib("id", "titlebox")
	titlebox.AddStyle("margin", "0 0 0 0")
	// Padding-top + height should be about 5em, padding decides the position
	titlebox.AddStyle("padding", "1.2em 0 0 1.8em")
	titlebox.AddStyle("height", "3.1em")
	titlebox.AddStyle("width", "100%")
	titlebox.AddStyle("position", "fixed")
	//titlebox.AddStyle("background-color", cs.Darkgray) // gray, could be a gradient
	if backgroundTextureURL != "" {
		titlebox.AddStyle("background", "url('"+backgroundTextureURL+"')")
	}
	//titlebox.AddStyle("z-index", "2") // 2 is above the search box which is 1

	if addSearchBox {
		searchbox := AddSearchBox(titlebox, searchURL, searchButtonText, roundedLook)
		searchbox.AddAttrib("id", "searchbox")
		searchbox.AddStyle("position", "relative")
		searchbox.AddStyle("float", "right")
		// The padding decides the position for this one
		searchbox.AddStyle("padding", "0.4em 3em 0 0")
		searchbox.AddStyle("margin", "0")
		//searchbox.AddStyle("min-width", "10em")
		//searchbox.AddStyle("line-height", "10em")
		//searchbox.AddStyle("z-index", "1") // below the title
	}

	return div, nil
}

// TODO: Place at the bottom of the content instead of at the bottom of the window
func AddFooter(page *onthefly.Page, footerText, footerTextColor, footerColor string, elapsed time.Duration) (*onthefly.Tag, error) {
	body, err := page.GetTag("body")
	if err != nil {
		return nil, err
	}
	div := body.AddNewTag("div")
	div.AddAttrib("id", "notice")
	div.AddStyle("position", "fixed")
	div.AddStyle("bottom", "0")
	div.AddStyle("left", "0")
	div.AddStyle("width", "100%")
	div.AddStyle("display", "block")
	div.AddStyle("padding", "0")
	div.AddStyle("margin", "0")
	div.AddStyle("background-color", footerColor)
	div.AddStyle("font-size", "0.6em")
	div.AddStyle("text-align", "right")
	div.AddStyle("box-shadow", "1px -2px 3px rgba(0,0,0, .5)")

	innerdiv := div.AddNewTag("div")
	innerdiv.AddAttrib("id", "innernotice")
	innerdiv.AddStyle("padding", "0 2em 0 0")
	innerdiv.AddStyle("margin", "0")
	innerdiv.AddStyle("color", footerTextColor)
	innerdiv.AddContent("Generated in " + elapsed.String() + " | " + footerText)

	return div, nil
}

func AddContent(page *onthefly.Page, contentTitle, contentHTML string) (*onthefly.Tag, error) {
	body, err := page.GetTag("body")
	if err != nil {
		return nil, err
	}

	div := body.AddNewTag("div")
	div.AddAttrib("id", "content")
	div.AddStyle("z-index", "-1")
	div.AddStyle("color", "black") // content headline color
	div.AddStyle("min-height", "80%")
	div.AddStyle("min-width", "60%")
	div.AddStyle("float", "left")
	div.AddStyle("position", "relative")
	div.AddStyle("margin-left", "4%")
	div.AddStyle("margin-top", "9.5em")
	div.AddStyle("margin-right", "5em")
	div.AddStyle("padding-left", "4em")
	div.AddStyle("padding-right", "5em")
	div.AddStyle("padding-top", "1em")
	div.AddStyle("padding-bottom", "2em")
	div.AddStyle("background-color", "rgba(255,255,255,0.92)")                                                                               // light gray. Transparency with rgba() doesn't work in IE
	div.AddStyle("filter", "progid:DXImageTransform.Microsoft.gradient(GradientType=0,startColorstr='#dcffffff', endColorstr='#dcffffff');") // for transparency in IE

	div.AddStyle("text-align", "justify")
	div.RoundedBox()

	h2 := div.AddNewTag("h2")
	h2.AddAttrib("id", "textheader")
	h2.AddContent(contentTitle)
	h2.CustomSansSerif("Armata")

	p := div.AddNewTag("p")
	p.AddAttrib("id", "textparagraph")
	p.AddStyle("margin-top", "0.5em")
	//p.CustomSansSerif("Junge")
	p.SansSerif()
	p.AddStyle("font-size", "1.0em")
	p.AddStyle("color", "black") // content text color
	p.AddContent(contentHTML)

	return div, nil
}

// Add a search box to the page, actionURL is the url to use as a get action,
// buttonText is the text on the search button
func AddSearchBox(tag *onthefly.Tag, actionURL, buttonText string, roundedLook bool) *onthefly.Tag {

	div := tag.AddNewTag("div")
	div.AddAttrib("id", "searchboxdiv")
	div.AddStyle("text-align", "right")
	div.AddStyle("display", "inline-block")

	form := div.AddNewTag("form")
	form.AddAttrib("id", "search")
	form.AddAttrib("method", "get")
	form.AddAttrib("action", actionURL)

	innerDiv := form.AddNewTag("div")
	innerDiv.AddAttrib("id", "innerdiv")
	innerDiv.AddStyle("overflow", "hidden")
	innerDiv.AddStyle("padding-right", "0.5em")
	innerDiv.AddStyle("display", "inline-block")

	inputText := innerDiv.AddNewTag("input")
	inputText.AddAttrib("id", "inputtext")
	inputText.AddAttrib("name", "q")
	inputText.AddAttrib("size", "40")
	inputText.AddStyle("padding", "0.25em")
	inputText.CustomSansSerif("Armata")
	inputText.AddStyle("background-color", "#f0f0f0")
	if roundedLook {
		inputText.RoundedBox()
	} else {
		inputText.AddStyle("border", "none")
	}

	// inputButton := form.AddNewTag("input")
	// inputButton.AddAttrib("id", "inputbutton")
	// // The position is in the margin
	// inputButton.AddStyle("margin", "0.08em 0 0 0.4em")
	// inputButton.AddStyle("padding", "0.2em 0.6em 0.2em 0.6em")
	// inputButton.AddStyle("float", "right")
	// inputButton.AddAttrib("type", "submit")
	// inputButton.AddAttrib("value", buttonText)
	// inputButton.SansSerif()
	// //inputButton.AddStyle("overflow", "hidden")
	// if roundedLook {
	// 	inputButton.RoundedBox()
	// } else {
	// 	inputButton.AddStyle("border", "none")
	// }

	return div
}

func AddTitleBox(tag *onthefly.Tag, title, subtitle string, cs *ColorScheme) *onthefly.Tag {

	div := tag.AddNewTag("div")
	div.AddAttrib("id", "titlebox")
	div.AddStyle("display", "block")
	div.AddStyle("position", "fixed")

	word1 := title
	word2 := ""
	if strings.Contains(title, " ") {
		word1 = strings.SplitN(title, " ", 2)[0]
		word2 = strings.SplitN(title, " ", 2)[1]
	}

	a := div.AddNewTag("a")
	a.AddAttrib("id", "homelink")
	a.AddAttrib("href", "/")
	a.AddStyle("text-decoration", "none")

	font0 := a.AddNewTag("div")
	font0.AddAttrib("id", "whitetitle")
	font0.AddAttrib("class", "titletext")
	font0.AddStyle("color", cs.TitleText)
	//font0.CustomSansSerif("Armata")
	font0.SansSerif()
	font0.AddStyle("font-size", "2.0em")
	font0.AddStyle("font-weight", "bolder")
	font0.AddContent(word1)

	font1 := a.AddNewTag("div")
	font1.AddAttrib("id", "bluetitle")
	font1.AddAttrib("class", "titletext")
	font1.AddStyle("color", cs.Nicecolor)
	//font1.CustomSansSerif("Armata")
	font1.SansSerif()
	font1.AddStyle("font-size", "2.0em")
	font1.AddStyle("font-weight", "bold")
	font1.AddStyle("overflow", "hidden")
	font1.AddContent(word2)

	font2 := a.AddNewTag("div")
	font2.AddAttrib("id", "graytitle")
	font2.AddAttrib("class", "titletext")
	font2.AddStyle("font-size", "0.5em")
	font2.AddStyle("color", "#707070")
	font2.CustomSansSerif("Armata")
	font2.AddStyle("font-size", "1.25em")
	font2.AddStyle("font-weight", "normal")
	font2.AddStyle("overflow", "hidden")
	font2.AddContent(subtitle)

	return div
}

// Takes a page and a colon-separated string slice of text:url, hiddenlinks is just a list of the url part
func AddMenuBox(page *onthefly.Page, darkBackgroundTexture string, customSansSerif string) (*onthefly.Tag, error) {
	body, err := page.GetTag("body")
	if err != nil {
		return nil, err
	}

	div := body.AddNewTag("div")
	div.AddAttrib("id", "menubox")
	div.AddStyle("display", "block")
	div.AddStyle("width", "100%")
	div.AddStyle("margin", "0")
	div.AddStyle("padding", "0.1em 0 0.2em 0")
	div.AddStyle("position", "absolute")
	div.AddStyle("top", "4.3em")
	div.AddStyle("left", "0")
	div.AddStyle("background-color", "#0c0c0c") // dark gray, fallback
	div.AddStyle("background", "url('"+darkBackgroundTexture+"')")
	div.AddStyle("position", "fixed")
	div.AddStyle("box-shadow", "1px 3px 5px rgba(0,0,0, .8)")
	if customSansSerif != "" {
		div.AddStyle("font-family", customSansSerif)
	}

	div.AddLastContent("{{{menu}}}")

	return div, nil
}
