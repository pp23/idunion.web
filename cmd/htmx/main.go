//go:generate oapi-codegen -package api -generate "chi-server,models" -o ../../internal/api/api.gen.go ../../api/openapi.yaml

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"htmx/internal/auth"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	api "htmx/internal/api"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
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

func (api *IduImpl) Register(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (api *IduImpl) GetToken(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (api *IduImpl) Login(w http.ResponseWriter, r *http.Request) {
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
	var authCode string
	URL, urlErr := url.ParseRequestURI("http://whoami.localhost:8085/auth")
	if urlErr != nil {
		log.Print(urlErr)
		return
	}
	res, err, statusCode := auth.RequestAuthCode(
		URL,
		url.UserPassword("example", "secret"),
		auth.Client{
			Client_id:    "idunion.web",
			Redirect_uri: "http://localhost:8080/", // TODO: redirect_uri should point to the destination URL of the app that wants to get accessed
		},
		"TODO",
	)
	if err != nil {
		log.Printf("ERROR: Login response body: %v", err)
		w.WriteHeader(statusCode)
		w.Write([]byte(fmt.Sprint(err)))
		return
	} else {
		log.Printf("StatusCode: %v", res.StatusCode)
		for k, v := range res.Header {
			log.Printf("%s: %s", k, v)
		}
		loc, locErr := res.Location()
		if locErr != nil {
			log.Printf("Error with response location: %v", locErr)
			return
		}
		log.Printf("Location: %s", loc)
		if authCodes, ok := loc.Query()["code"]; ok {
			if len(authCodes) == 1 { // only one code query parameter accepted
				authCode = authCodes[0]
			} else {
				log.Printf("ERROR: Not exactly 1 code parameter found")
				return
			}
		} else {
			log.Printf("Location %s contains no code query parameter", loc)
			return
		}
		data, bodyErr := ioutil.ReadAll(res.Body)
		if bodyErr != nil {
			log.Printf("Could not read response payload: %v", bodyErr)
		}
		log.Printf("data: %v", string(data))
	}
	// get token
	{
		URL, urlErr := url.ParseRequestURI("http://whoami.localhost:8085/token")
		if urlErr != nil {
			log.Print(urlErr)
			return
		}
		log.Printf("AuthCode: %s", authCode)
		query := url.Values{
			"redirect_uri": []string{"http://localhost:8080/"},
			"grant_type":   []string{"authorization_code"},
			"code":         []string{authCode},
			"client_id":    []string{"idunion.web"},
		}
		URL.RawQuery = query.Encode()
		log.Printf("Token-URL: %s", URL.String())
		req, reqErr := http.NewRequest("POST", URL.String(), nil)
		if reqErr != nil {
			log.Printf("ERROR: Could not construct token request: %v", reqErr)
			return
		}
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			log.Printf("ERROR obtaining token: %v", err)
			return
		}
		data, bodyErr := ioutil.ReadAll(res.Body)
		if bodyErr != nil {
			log.Printf("ERROR: Could not resd token response: %v", bodyErr)
			return
		}
		log.Printf("Token response: %s", string(data))
		type Token struct {
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    uint64 `json:"expires_in"`
			RefreshToken string `json:"refresh_token,omitempty"`
		}
		var jToken Token
		errJson := json.Unmarshal(data, &jToken)
		if errJson != nil {
			log.Printf("ERROR: JSON unmarshalling error: %v", errJson)
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
		w.WriteHeader(http.StatusOK)
		return
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
