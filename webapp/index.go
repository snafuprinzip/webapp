package webapp

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func HandleHome(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Display Home Page
	RenderTemplate(w, r, "index/home", map[string]interface{}{
		"Pagetitle": "WebApp",
	})
}
