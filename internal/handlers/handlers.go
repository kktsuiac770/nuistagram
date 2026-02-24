package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

var templates *template.Template

var funcMap = template.FuncMap{
	"sub": func(a, b int) int { return a - b },
	"add": func(a, b int) int { return a + b },
}

func InitTemplates() error {
	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	return err
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	err := templates.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type PageData struct {
	User  interface{}
	Data  interface{}
	Title string
}

func Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	user := GetCurrentUser(r)

	tagsParam := r.URL.Query().Get("tags")
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "or"
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	var tags []string
	if tagsParam != "" {
		for _, tag := range strings.Split(tagsParam, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	var userID int64
	if user != nil {
		userID = user.ID
	}

	result, err := SearchPhotos(tags, mode, page, userID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	nuis, err := GetAllNuis()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	users, _ := GetAllUsers()

	renderTemplate(w, "index.html", map[string]interface{}{
		"User":         user,
		"Photos":       result.Photos,
		"Nuis":         nuis,
		"Users":        users,
		"SelectedTags": tags,
		"Mode":         mode,
		"Pagination":   result,
	})
}
