package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

// Define a global variable for the templates
var templates = template.Must(template.ParseFiles(
	"templates/index.html",
	"templates/home.html",  // Add this if you have a separate home page template
	"templates/about.html", // Similarly for other pages
	"templates/contact.html",
	"templates/faq.html",
	"templates/user/login.html", // template.ParseFiles takes only the basename
))

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
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
