package views

import (
	"embed"
	"fmt"
	"html/template"
)

//go:embed *.tmpl
var FS embed.FS

func Parse(templateFiles ...string) (*template.Template, error) {
	files := append([]string{"layout.tmpl"}, templateFiles...)
	tmpl, err := template.ParseFS(FS, files...)
	if err != nil {
		return nil, fmt.Errorf("parse templates %v: %w", files, err)
	}
	return tmpl, nil
}
