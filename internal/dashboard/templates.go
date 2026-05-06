package dashboard

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"
)

//go:embed templates/*.tmpl static/*
var assets embed.FS

func parseTemplates() (*template.Template, error) {
	tmpl, err := template.New("dashboard").ParseFS(assets, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse dashboard templates: %w", err)
	}
	return tmpl, nil
}

func staticFS() (fs.FS, error) {
	sub, err := fs.Sub(assets, path.Clean("static"))
	if err != nil {
		return nil, fmt.Errorf("load dashboard static assets: %w", err)
	}
	return sub, nil
}
