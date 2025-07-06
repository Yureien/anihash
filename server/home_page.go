package server

import (
	"html/template"
	"net/http"
)

func (s server) homePageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("server/templates/home.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, nil)
}
