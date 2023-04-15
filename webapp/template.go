package webapp

import (
	"bytes"
	"fmt"
	"github.com/snafuprinzip/webappskeleton"
	"html/template"
	"net/http"
)

var layoutFuncs = template.FuncMap{
	"yield": func() (string, error) {
		return "", fmt.Errorf("yield called inappropriately")
	},
}

// layout template will be used every time as a wrapper around the content of the other
// page templates
var layout = template.Must(
	template.
		New("layout.html").
		Funcs(layoutFuncs).
		ParseFiles("templates/layout.html"),
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
var templates = template.Must(template.New("t").ParseGlob("templates/**/*.html"))

// RenderTemplate picks a template by "path/name" as defined within the template and renders it for the
// client browser
func RenderTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}

	data["CurrentUser"] = RequestUser(r)
	data["OpenRegistration"] = webappskeleton.Config.OpenRegistration
	data["Flash"] = r.URL.Query().Get("flash")

	funcs := template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf := bytes.NewBuffer(nil)
			err := templates.ExecuteTemplate(buf, name, data)
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
