package webapp

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

var layoutFuncs = template.FuncMap{
	"yield": func() (string, error) {
		return "", fmt.Errorf("yield called inappropriately")
	},
}
var availableLanguages []string

// layout template will be used every time as a wrapper around the content of the other
// page templates
var layout = template.Must(
	template.New("layout.html").Funcs(layoutFuncs).ParseFiles("templates/layout.html"),
)

// errorTemplate is a small web page showing the template rendering error if any occur
var errorTemplate = `
<html>
	<body>
		<h1>Error rendering template %s</h1>
		<p>%s</p>
	</body>
</html>
`

// templates is a collection of all files in a subdirectory of the templates/ directory
var templates = template.Must(template.New("t").ParseGlob("templates/??/**/*.html"))

// GetLanguageFromUser reads the given user's config and returns the language if it is available or "en" for english
func GetLanguageFromUser(userid string) (string, error) {
	var languageError string
	var lang = "en"

	conf, err := GlobalUserConfigStore.Find(userid)
	if err != nil {
		log.Fatalln("Unable to read from global userconfigs store:", err)
	}
	if conf != nil {
		if IsLanguageAvailable(conf.Language) {
			lang = conf.Language
		} else {
			languageError = "Configured language " + conf.Language + " is not available. Sorry!"
			return "en", errors.New(languageError)
		}
	}

	return lang, nil
}

// GetLanguageFromRequest reads the user's config from the current session and returns the language if it is available
// or "en" for english
func GetLanguageFromRequest(r *http.Request) (string, error) {
	var lang = "en"

	if r != nil {
		user := RequestUser(r)
		if user != nil {
			conf, err := GlobalUserConfigStore.Find(user.ID)
			if err != nil {
				log.Fatalln("Unable to read from global userconfigs store:", err)
			}
			if conf != nil {
				if IsLanguageAvailable(conf.Language) {
					lang = conf.Language
				} else {
					return "en", errors.New("Configured language " + conf.Language + " is not available. Sorry!")
				}
			}
		}
	}

	if r.URL.Query().Has("lang") {
		l := r.URL.Query().Get("lang")[:2]
		if IsLanguageAvailable(l) {
			lang = l
		} else {
			return "en", errors.New("Requested language " + l + " is not available. Sorry!")
		}
	}
	return lang, nil
}

// RenderTemplate picks a template by "path/name" as defined within the template and renders it for the
// client browser
func RenderTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}

	var languageError string
	language, err := GetLanguageFromRequest(r)
	if err != nil {
		languageError = fmt.Sprintf("%s", err)
	}

	user := RequestUser(r)
	data["CurrentUser"] = user
	data["OpenRegistration"] = Config.OpenRegistration
	data["Flash"] = r.URL.Query().Get("flash") + languageError
	data["Language"] = language
	data["AdminAccount"] = IsAdmin(r)
	data["AppName"] = appName

	funcs := template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf := bytes.NewBuffer(nil)
			err := templates.ExecuteTemplate(buf, language+"/"+name, data)
			return template.HTML(buf.String()), err
		},
	}

	layoutClone, _ := layout.Clone()
	layoutClone.Funcs(funcs)
	err = layoutClone.Execute(w, data)

	if err != nil {
		http.Error(
			w,
			fmt.Sprintf(errorTemplate, name, err),
			http.StatusInternalServerError,
		)
	}
}

func IsLanguageAvailable(language string) bool {
	// initialize availableLanguages for caching if needed
	// from available templates
	if availableLanguages == nil {
		entries, err := os.ReadDir("templates")
		if err != nil {
			log.Fatalln("Unable to access templates directory:", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				availableLanguages = append(availableLanguages, e.Name())
			}
		}
	}

	for _, lang := range availableLanguages {
		if lang == language {
			return true
		}
	}

	return false
}
