package rest

import (
	"github.com/boggydigital/nod"
	"io"
	"net/http"
)

func GetHealth(w http.ResponseWriter, _ *http.Request) {

	// GET /health

	if err := AccessControlAllowOrigin(w); err != nil {
		http.Error(w, nod.Error(err).Error(), http.StatusInternalServerError)
		return
	}

	if _, err := io.WriteString(w, "ok"); err != nil {
		http.Error(w, nod.Error(err).Error(), http.StatusInternalServerError)
		return
	}
}
