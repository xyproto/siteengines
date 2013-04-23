package siteengines

import (
	. "github.com/xyproto/browserspeak"
	"github.com/xyproto/genericsite"
	"github.com/xyproto/instapage"
	"github.com/xyproto/simpleredis"
	"github.com/xyproto/web"
)

type IPEngine struct {
	userState *genericsite.UserState
	data      *simpleredis.List
}

func NewIPEngine(userState *genericsite.UserState) *IPEngine {

	pool := userState.GetPool()

	// Create a RedisList for storing IP adresses
	ips := simpleredis.NewList(pool, "IPs")

	ipEngine := new(IPEngine)
	ipEngine.data = ips
	ipEngine.userState = userState

	return ipEngine
}

// Set an IP adress and generate a confirmation page for it
func (ie *IPEngine) GenerateSetIP() WebHandle {
	return func(ctx *web.Context, val string) string {
		if val == "" {
			return "Empty value, IP not set"
		}
		ie.data.Add(val)
		return "OK, set IP to " + val
	}
}

// Get all the stored IP adresses and generate a page for it
func (ie *IPEngine) GenerateGetAllIPs() WebHandle {
	return func(ctx *web.Context, val string) string {
		username := genericsite.GetBrowserUsername(ctx)
		if username == "" {
			return "No user logged in"
		}
		if !ie.userState.IsLoggedIn(username) {
			return "Not logged in"
		}
		s := ""
		iplist, err := ie.data.GetAll()
		if err == nil {
			for _, val := range iplist {
				s += "IP: " + val + "<br />"
			}
		}
		return instapage.Message("IPs", s)
	}
}

// Get the last stored IP adress and generate a page for it
func (ie *IPEngine) GenerateGetLastIP() WebHandle {
	return func(ctx *web.Context, val string) string {
		username := genericsite.GetBrowserUsername(ctx)
		if username == "" {
			return "No user logged in"
		}
		if !ie.userState.IsLoggedIn(username) {
			return "Not logged in"
		}
		s := ""
		ip, err := ie.data.GetLast()
		if err == nil {
			s = "IP: " + ip
		}
		return s
	}
}

func (ie *IPEngine) ServePages() {
	// TODO: REST service instead
	web.Get("/setip/(.*)", ie.GenerateSetIP())
	web.Get("/getip/(.*)", ie.GenerateGetLastIP())
	web.Get("/getallips/(.*)", ie.GenerateGetAllIPs())
}
