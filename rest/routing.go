package rest

import (
	"github.com/boggydigital/nod"
	"net/http"
)

var (
	Log      = nod.RequestLog
	Redirect = http.RedirectHandler
)

func HandleFuncs() {

	patternHandlers := map[string]http.Handler{
		// static resources
		"GET /health":     Log(http.HandlerFunc(GetHealth)),
		"OPTIONS /health": Log(http.HandlerFunc(GetHealth)),
	}

	for p, h := range patternHandlers {
		http.HandleFunc(p, h.ServeHTTP)
	}
}
