package siteengines

import (
	"github.com/hoisie/web"
	"github.com/xyproto/pinterface"
	. "github.com/xyproto/webhandle"
)

type IPEngine struct {
	state pinterface.IUserState
	data  pinterface.IList
}

func NewIPEngine(userState pinterface.IUserState) (*IPEngine, error) {

	creator := userState.Creator()

	// Create a RedisList for storing IP adresses
	if ips, err := creator.NewList("IPs"); err != nil {
		return nil, err
	} else {
		ipEngine := new(IPEngine)
		ipEngine.state = userState
		ipEngine.data = ips
		return ipEngine, nil
	}
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
		username := ie.state.Username(ctx.Request)
		if username == "" {
			return "No user logged in"
		}
		if !ie.state.IsLoggedIn(username) {
			return "Not logged in"
		}
		s := ""
		iplist, err := ie.data.GetAll()
		if err == nil {
			for _, val := range iplist {
				s += "IP: " + val + "<br />"
			}
		}
		return Message("IPs", s)
	}
}

// Get the last stored IP adress and generate a page for it
func (ie *IPEngine) GenerateGetLastIP() WebHandle {
	return func(ctx *web.Context, val string) string {
		username := ie.state.Username(ctx.Request)
		if username == "" {
			return "No user logged in"
		}
		if !ie.state.IsLoggedIn(username) {
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
