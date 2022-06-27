package api

import (
	_ "embed"
	"html/template"
	"log"
)

//go:embed index.html
var indexTemplate string

func buildTemplate() *template.Template {
	html, err := template.New("index.html").Parse(indexTemplate)
	if err != nil {
		log.Fatal(err)
	}
	return html
}
