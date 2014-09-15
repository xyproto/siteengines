package siteengines

import (
	"net/http"

	"github.com/xyproto/instapage"
	"github.com/xyproto/permissions"
	"github.com/xyproto/simpleredis"
	. "github.com/xyproto/webhandle"
)

type IPEngine struct {
	state *permissions.UserState
	data  *simpleredis.List
}

func NewIPEngine(state *permissions.UserState) *IPEngine {

	// Create a RedisList for storing IP adresses
	ips := simpleredis.NewList(state.GetPool(), "IPs")

	ipEngine := new(IPEngine)
	ipEngine.data = ips
	ipEngine.state = state

	return ipEngine
}

// Set an IP adress and generate a confirmation page for it
func (ie *IPEngine) GenerateSetIP() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ip := GetLast(req.URL)
		if ip == "" {
			Ret(w, "Empty value, IP not set")
			return
		}
		ie.data.Add(ip)
		Ret(w, "OK, set IP to "+ip)
	}
}

// Get all the stored IP adresses and generate a page for it
func (ie *IPEngine) GenerateGetAllIPs() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		username := ie.state.GetUsername(req)
		if username == "" {
			Ret(w, "No user logged in")
			return
		}
		if !ie.state.IsLoggedIn(username) {
			Ret(w, "Not logged in")
			return
		}
		s := ""
		iplist, err := ie.data.GetAll()
		if err == nil {
			for _, val := range iplist {
				s += "IP: " + val + "<br />"
			}
		}
		Ret(w, instapage.Message("IPs", s))
	}
}

// Get the last stored IP adress and generate a page for it
func (ie *IPEngine) GenerateGetLastIP() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		username := ie.state.GetUsername(req)
		if username == "" {
			Ret(w, "No user logged in")
			return
		}
		if !ie.state.IsLoggedIn(username) {
			Ret(w, "Not logged in")
			return
		}
		s := ""
		ip, err := ie.data.GetLast()
		if err == nil {
			s = "IP: " + ip
		}
		Ret(w, s)
	}
}

func (ie *IPEngine) ServePages(mux *http.ServeMux) {
	// TODO: REST service instead
	mux.HandleFunc("/setip/", ie.GenerateSetIP())
	mux.HandleFunc("/getip/", ie.GenerateGetLastIP())
	mux.HandleFunc("/getallips/", ie.GenerateGetAllIPs())
}
