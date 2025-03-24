//go:generate oapi-codegen -package api -generate "chi-server,models" -o ../../internal/api/api.gen.go ../../api/openapi.yaml

package main

import (
	"fmt"
	"html/template"
	"htmx/internal/auth/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	api "htmx/internal/api"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	"gopkg.in/validator.v2"
)

type IduImpl struct {
	store *sessions.CookieStore
}

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
	"MAIL_DOMAIN":               "idunion.me",
	"USERNAME_VALIDATION_REGEX": "/^[a-z0-9._-]+$/",
}

type UserCredentials struct {
	Username string `validate:"min=3,max=30,regexp=^[a-z0-9._-]+$`
	Password string `validate:"min=6"`
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

		if template, err := ioutil.ReadFile(f); err == nil {
			content := string(template)
			for param, value := range staticParams {
				content = strings.ReplaceAll(content, "[["+param+"]]", value)
			}
			_, err := renderedFile.WriteString(content)
			if err != nil {
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

func (api *IduImpl) Register(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (api *IduImpl) GetToken(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (api *IduImpl) Login(w http.ResponseWriter, r *http.Request) {
	// read form values
	UserCredentials := UserCredentials{
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}
	if err := validator.Validate(UserCredentials); err != nil {
		// TODO: Do not print the password.
		log.Printf("Validationerror: %s. Values: Username: %s Password: %s", err, UserCredentials.Username, UserCredentials.Password)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Credentials invalid"))
		return
	}
	// check for cookie
	for _, cookie := range r.Cookies() {
		log.Printf("COOKIE: %s  VALUE: %s", cookie.Name, cookie.Value)
	}
	{
		token, err := api.store.Get(r, "t")
		if err != nil {
			log.Printf("ERROR: Could not get token: %v", err)
		}
		log.Printf("TOKEN: %v", token.Values)
	}
	// get auth code
	URL, urlErr := url.ParseRequestURI("http://whoami.localhost:8085/auth")
	if urlErr != nil {
		log.Print(urlErr)
		return
	}
	client := oauth2.Client{
		Client_id:    "idunion.web",
		Redirect_uri: "http://localhost:8080/", // TODO: redirect_uri should point to the destination URL of the app that wants to get accessed
	}
	res, err := oauth2.RequestAuthCode(
		&oauth2.AuthCodeRequest{
			AuthUrl: URL,
			Client:  &client,
			Scope:   "TODO: scope",
		},
		url.UserPassword(UserCredentials.Username, UserCredentials.Password),
		"TODO: state",
		"TODO: code_challenge",
	)
	if err != nil {
		log.Printf("ERROR: Getting authCode: %v", err)
		w.WriteHeader(res.StatusCode)
		w.Write([]byte(fmt.Sprint(err)))
		return
	} else {
		log.Printf("StatusCode: %v", res.StatusCode)
	}
	// get token
	tokenUrl, tokenUrlErr := url.ParseRequestURI("http://whoami.localhost:8085/token")
	if tokenUrlErr != nil {
		log.Print(tokenUrlErr)
		return
	}
	jToken, tokenErr := oauth2.RequestToken(&oauth2.TokenRequest{
		TokenUrl:         tokenUrl,
		AuthCodeResponse: res,
		Client:           &client,
	})
	if tokenErr != nil {
		log.Printf("ERROR: Could not obtain token: %v", tokenErr)
		return
	}
	// Wrap token into cookie and send it back
	session, errSession := api.store.Get(r, "t")
	if errSession != nil {
		log.Printf("WARNING: Get session error: %v", errSession)
	}
	session.Options.Secure = true
	session.Options.HttpOnly = true // prevent Javascript access
	session.Options.SameSite = http.SameSiteStrictMode
	session.Options.MaxAge = 0 // 0: last until session end
	// cookie values
	session.Values["at"] = jToken.AccessToken
	session.Values["rt"] = jToken.RefreshToken
	session.Values["exp"] = jToken.ExpiresIn
	if err := session.Save(r, w); err != nil {
		log.Printf("ERROR: Could not save session: %v", err)
		return
	}
	http.Redirect(w, r, "/about", http.StatusTemporaryRedirect)
	return
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
	iduImpl := &IduImpl{
		store: sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY"))),
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Mount("/", api.HandlerWithOptions(iduImpl, chiServerOptions))
	http.Handle("/api/", r) // trailing "/" is a wildcard pattern for subsequent segments

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
