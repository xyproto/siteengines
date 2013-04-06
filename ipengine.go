package siteengines

import (
	. "github.com/xyproto/browserspeak"
	"github.com/xyproto/genericsite"
	"github.com/xyproto/instapage"
	"github.com/xyproto/simpleredis"
	"github.com/xyproto/web"
)

type IPEngine struct {
	state *genericsite.UserState
	data  *simpleredis.List
}

func NewIPEngine(state *genericsite.UserState) *IPEngine {

	pool := state.GetPool()

	// Create a RedisList for storing IP adresses
	ips := simpleredis.NewList(pool, "IPs")

	ipEngine := new(IPEngine)
	ipEngine.data = ips
	ipEngine.state = state

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
func (ie *IPEngine) GenerateGetAllIPs() SimpleWebHandle {
	return func(val string) string {
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
func (ie *IPEngine) GenerateGetLastIP() SimpleWebHandle {
	return func(val string) string {
		s := ""
		ip, err := ie.data.GetLast()
		if err == nil {
			s = "IP: " + ip
		}
		return s
	}
}

func (ie *IPEngine) ServePages() {
	web.Get("/setip/(.*)", ie.GenerateSetIP())
	web.Get("/getip/(.*)", ie.GenerateGetLastIP())
	web.Get("/getallips/(.*)", ie.GenerateGetAllIPs())
}
