package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/timcki/switter/pkg/users"
)

func main() {

	r := mux.NewRouter()

	neo4jUri, found := os.LookupEnv("NEO4J_URI")
	if !found {
		panic("NEO4J_URI not set")
	}
	neo4jUsername, found := os.LookupEnv("NEO4J_USERNAME")
	if !found {
		panic("NEO4J_USERNAME not set")
	}
	neo4jPassword, found := os.LookupEnv("NEO4J_PASSWORD")
	if !found {
		panic("NEO4J_PASSWORD not set")
	}

	usersRepository := users.UserNeo4jRepository{
		Driver: driver(neo4jUri, neo4j.BasicAuth(neo4jUsername, neo4jPassword, "")),
	}
	registrationHandler := &users.UserRegistrationHandler{
		Path:           "/users",
		UserRepository: &usersRepository,
	}
	loginHandler := &users.UserLoginHandler{
		Path:           "/users/login",
		UserRepository: &usersRepository,
	}

	basePath := filepath.Join("tmpl", "base.html")

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Do stuff here
			log.Println(r.RequestURI)
			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r)
		})
	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmplPath := filepath.Join("tmpl", "index.html")
		tmpl := template.Must(template.ParseFiles(tmplPath, basePath))
		tmpl.ExecuteTemplate(w, "base", "")
	})
	r.HandleFunc("/login_page", func(w http.ResponseWriter, r *http.Request) {
		tmplPath := filepath.Join("tmpl", "login.html")
		tmpl := template.Must(template.ParseFiles(tmplPath, basePath))
		tmpl.ExecuteTemplate(w, "base", "")
	})
	r.HandleFunc("/register_page", func(w http.ResponseWriter, r *http.Request) {
		tmplPath := filepath.Join("tmpl", "register.html")
		tmpl := template.Must(template.ParseFiles(tmplPath, basePath))
		tmpl.ExecuteTemplate(w, "base", "")
	})
	r.HandleFunc(registrationHandler.Path, registrationHandler.Register)
	r.HandleFunc(loginHandler.Path, loginHandler.Login)

	if err := http.ListenAndServe(":3000", r); err != nil {
		panic(err)
	}
}

func driver(target string, token neo4j.AuthToken) neo4j.Driver {
	result, err := neo4j.NewDriver(target, token)
	if err != nil {
		panic(err)
	}
	return result
}
