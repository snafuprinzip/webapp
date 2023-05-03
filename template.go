package webapp

import (
	"bytes"
	"fmt"
	"html/template"
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

// RenderTemplate picks a template by "path/name" as defined within the template and renders it for the
// client browser
func RenderTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	var languageError string
	var darkmode string = "light"
	var conf *UserConfig

	if data == nil {
		data = map[string]interface{}{}
	}

	user := RequestUser(r)
	if user != nil {
		conf, _ = GlobalUserConfigStore.Find(RequestUser(r).ID)
		if conf != nil {
			if conf.DarkMode {
				darkmode = "dark"
			}
		}
	}
	lang := GetLanguage("", r, nil)

	data["CurrentUser"] = RequestUser(r)
	data["OpenRegistration"] = Config.OpenRegistration
	data["Flash"] = r.URL.Query().Get("flash") + languageError
	data["Language"] = lang
	data["AdminAccount"] = IsAdmin(r)
	data["AppName"] = appName
	data["DarkMode"] = darkmode

	idx := fmt.Sprintf("%v", data["Pagetitle"])
	data["Headline"] = LookupTranslation(r, "Title"+idx)

	funcs := template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf := bytes.NewBuffer(nil)
			err := templates.ExecuteTemplate(buf, lang+"/"+name, data)
			return template.HTML(buf.String()), err
		},
	}

	layoutClone, _ := layout.Clone()
	layoutClone.Funcs(funcs)
	err := layoutClone.Execute(w, data)
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
			Logln(FatalLevel, "Unable to access templates directory:", err)
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
