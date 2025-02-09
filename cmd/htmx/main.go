//go:generate oapi-codegen -package api -generate "chi-server,models" -o ../../internal/api/api.gen.go ../../api/openapi.yaml

package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	api "htmx/internal/api"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type IduImpl struct{}

var chiServerOptions = api.ChiServerOptions{
	// the frontend forwards api/ requests to the api-handling service
	// and uses the defined api-version of the backend, so no api/v0 required
	BaseURL: "/api",
}

// holds the templates after rendering static values
var templates *template.Template

// Define a global variable for the templates
var templateFiles = []string{
	"templates/index.html",
	"templates/home.html",  // Add this if you have a separate home page template
	"templates/about.html", // Similarly for other pages
	"templates/contact.html",
	"templates/faq.html",
	"templates/user/login.html", // template.ParseFiles takes only the basename
}

var staticTemplateParams = map[string]string{
	"USERNAME_VALIDATION_REGEX": "/^[a-z0-9._-]+$/",
}

// renders template files and replaces the static parameters
// usable if parameters/const values in the templates rely on configuration of this application
func staticRenderTemplates(templateFiles []string, staticParams map[string]string, outDir string) error {
	for _, f := range templateFiles {
		renderedFilename := filepath.Join(outDir, f)
		if err := os.MkdirAll(filepath.Dir(renderedFilename), 0700); err != nil {
			return err
		}
		renderedFile, fErr := os.OpenFile(renderedFilename, os.O_CREATE|os.O_WRONLY, 0700)
		if fErr != nil {
			return fErr
		}
		defer renderedFile.Close() // backup. intentionally ignore error as we call close explicitly

		if template, err := template.ParseFiles(f); err == nil {
			if err := template.Execute(renderedFile, staticParams); err != nil {
				renderedFile.Close()
				return err
			}
		} else {
			renderedFile.Close()
			return err
		}

		if err := renderedFile.Close(); err != nil {
			return err
		}
	}
	return nil
}

// IndexHandler serves the home page
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", nil)
	if err != nil {
		http.Error(w, "Error rendering home page", http.StatusInternalServerError)
		log.Printf("Error rendering home page: %v", err)
	}
}

// PageHandler serves other static pages
func PageHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path)
	page := r.URL.Path[1:] // Remove leading '/'
	if page == "" {
		page = "home"
	}
	err := templates.ExecuteTemplate(w, page+".html", nil)
	if err != nil {
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
		log.Printf("Error rendering page: %v", err)
	}
}

func main() {
	log.SetOutput(os.Stdout)                             // Log to standard output
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Include date, time, and file info

	outDir, tmpDirErr := os.MkdirTemp("/tmp", "rendered_http_templates")
	if tmpDirErr != nil {
		log.Fatal(tmpDirErr)
	}
	if err := staticRenderTemplates(templateFiles, staticTemplateParams, outDir); err != nil {
		log.Fatal(err)
	}
	var newTemplateFiles []string
	for _, f := range templateFiles {
		newTemplateFiles = append(newTemplateFiles, filepath.Join(outDir, f))
	}
	var templateErr error
	templates, templateErr = template.ParseFiles(newTemplateFiles...)
	if templateErr != nil {
		log.Fatal(templateErr)
	}
	// not too bad if the temporary dir could not get removed
	if err := os.RemoveAll(outDir); err != nil {
		log.Printf("ERROR: Could not remove temporary template directory %s: %v", outDir, err)
	}

	// Serve static files from the "static" directory
	staticDir := "./static"
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	http.HandleFunc("/", IndexHandler)

	// Serve other pages
	http.HandleFunc("/home", PageHandler)
	http.HandleFunc("/about", PageHandler)
	http.HandleFunc("/contact", PageHandler)
	http.HandleFunc("/faq", PageHandler)
	// Serve user related fragments
	http.HandleFunc("/login", PageHandler)

	log.Println("Starting server on :8080")
	var iduImpl api.Unimplemented
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Mount("/api/v0", api.HandlerWithOptions(&iduImpl, chiServerOptions))
	http.Handle("/api/v0", r)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
