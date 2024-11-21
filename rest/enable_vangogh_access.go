package rest

import (
	"github.com/arelate/theo/data"
	"net/http"
)

func AccessControlAllowOrigin(w http.ResponseWriter) (err error) {
	if rdx, err = rdx.RefreshReader(); err != nil {
		return err
	}

	if vangoghUrl, err := data.VangoghUrl(rdx, "", nil); err == nil {
		w.Header().Add("Access-Control-Allow-Origin", vangoghUrl.String())
		w.Header().Add("Access-Control-Allow-Private-Network", "true")
		return nil
	} else {
		return err
	}
}
