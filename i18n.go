package webapp

import (
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
)

var bundle *i18n.Bundle

func LookupTranslation(r *http.Request, msgid string) string {
	lang := r.FormValue("lang")
	accept := r.Header.Get("Accept-Language")
	prefs := GetLanguage("", r, nil)
	localizer := i18n.NewLocalizer(bundle, lang, prefs, accept)

	res, _ := localizer.Localize(&i18n.LocalizeConfig{MessageID: msgid})
	return res
}

func LookupTranslationWithData(r *http.Request, msgid string, data map[string]interface{}, count int) string {
	lang := r.FormValue("lang")
	accept := r.Header.Get("Accept-Language")
	prefs := GetLanguage("", r, nil)
	localizer := i18n.NewLocalizer(bundle, lang, prefs, accept)

	res, _ := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: msgid,
		DefaultMessage: &i18n.Message{
			ID: msgid,
		},
		TemplateData: data,
		PluralCount:  count,
	})
	return res
}

func LookupComplexTranslation(r *http.Request, msgid string, data map[string]interface{}, count int, funcs template.FuncMap) string {
	lang := r.FormValue("lang")
	accept := r.Header.Get("Accept-Language")
	prefs := GetLanguage("", r, nil)
	localizer := i18n.NewLocalizer(bundle, lang, prefs, accept)

	res, _ := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: msgid,
		DefaultMessage: &i18n.Message{
			ID: msgid,
		},
		TemplateData: data,
		PluralCount:  count,
		Funcs:        funcs,
	})
	return res
}

func SetupTranslations() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	dir := path.Join(Config.DataDirectory, "i18n")
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Println("Unable to read translation files:", err)
		return
	}
	for _, file := range files {
		if _, err := bundle.LoadMessageFile(path.Join(dir, file.Name())); err != nil {
			log.Println("goi18n: error loading translation file:", err)
		}
	}
}
