package server

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates/home.html
var homeTemplateFS embed.FS

var (
	homeTemplate = template.Must(template.ParseFS(homeTemplateFS, "templates/home.html"))
)

type homePageData struct {
	Host string
}

func (s server) homePageHandler(w http.ResponseWriter, r *http.Request) {
	data := homePageData{
		Host: r.Host,
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	err := homeTemplate.ExecuteTemplate(w, "home.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
