package webapp

import (
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"path"
)

var appName string

func NewApp(name string) {
	appName = name

	if err := os.MkdirAll(path.Join(Config.DataDirectory, "i18n"), 0750); err != nil {
		log.Println("error creating data directory", Config.DataDirectory, err)
	}
	if err := os.MkdirAll(Config.LogDirectory, 0750); err != nil {
		log.Println("error creating log directory", Config.LogDirectory, err)
	}

	// set up multi-language support
	SetupTranslations()

	SetupLogging()
}

func HandleHome(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Display Home Page
	RenderTemplate(w, r, "index/home", map[string]interface{}{
		"Pagetitle": "Main",
		"Welcome":   LookupTranslation(r, "welcome"),
	})
	//fmt.Println(LookupTranslation(r, "cats", map[string]interface{}{
	//	"Name":  "Lemmy",
	//	"Count": 2,
	//},
	//	2,
	//))
}
