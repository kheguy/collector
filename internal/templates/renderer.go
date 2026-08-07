package templates

import (
	"html/template"
	"net/http"
)

type TemplateRenderer struct {
	templates map[string]*template.Template
}

func MakeNewTemplateRenderer() (*TemplateRenderer, error) {
	templates := make(map[string]*template.Template)

	tmpl, err := template.ParseFiles(
		"internal/templates/layout.html",
		"internal/templates/list.html",
	)
	if err != nil {
		return nil, err
	}

	templates["list"] = tmpl

	return &TemplateRenderer{
		templates: templates,
	}, nil
}

func (r *TemplateRenderer) Render(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := r.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
