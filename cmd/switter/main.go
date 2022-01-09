package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	//"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/timcki/switter/internal/post"
	"github.com/timcki/switter/internal/users"
)

func driver(target string, token neo4j.AuthToken) neo4j.Driver {
	result, err := neo4j.NewDriver(target, token)
	if err != nil {
		panic(err)
	}
	return result
}

func getNeo4j() *users.UserNeo4jRepository {
	var uri, user, password string
	var found bool

	if uri, found = os.LookupEnv("NEO4J_URI"); !found {
		panic("NEO4J_URI not set")
	}
	if user, found = os.LookupEnv("NEO4J_USERNAME"); !found {
		panic("NEO4J_USERNAME not set")
	}
	if password, found = os.LookupEnv("NEO4J_PASSWORD"); !found {
		panic("NEO4J_PASSWORD not set")
	}

	return &users.UserNeo4jRepository{Driver: driver(uri, neo4j.BasicAuth(user, password, ""))}
}

func parseJwt(r *http.Request) (string, error) {
	token, err := r.Cookie("jwt_token")
	if err != nil {
		return "", err
	}
	return token.Value, nil
}

func renderTemplateHandler(path string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		tmplPath := filepath.Join("tmpl", path)
		tmpl := template.Must(template.ParseFiles(tmplPath, tmplBasePath))
		tmpl.ExecuteTemplate(w, "base", "")
	}
}

var tmplBasePath = filepath.Join("tmpl", "base.html")

func main() {

	r := mux.NewRouter()
	auth := r.PathPrefix("/auth").Subrouter()
	u := r.PathPrefix("/users").Subrouter()

	repo := getNeo4j()

	registrationHandler := &users.UserRegistrationHandler{Path: "/register", UserRepository: repo}
	loginHandler := &users.UserLoginHandler{Path: "/login", UserRepository: repo}

	// Middlewares
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.Method, r.RequestURI)
			next.ServeHTTP(w, r)
		})
	})

	auth.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			j, err := parseJwt(r)
			if err != nil {
				http.Redirect(w, r, "/register", http.StatusTemporaryRedirect)
				return
			}

			/*
				token, err := jwt.ParseWithClaims(
					j, &users.CustomClaims{},
					func(token *jwt.Token) (interface{}, error) {
						return []byte(os.Getenv("JWT_KEY")), nil
					})

				if err != nil {
					http.Redirect(w, r, "/register", http.StatusTemporaryRedirect)
					return
				}

				claims, ok := token.Claims.(*users.CustomClaims)
				if !ok {
					http.Redirect(w, r, "/register", http.StatusTemporaryRedirect)
					return
				}
			*/
			//r.Header["username"] = []string{j}
			r.Header.Add("username", j)
			next.ServeHTTP(w, r)
		})
	})

	// Regular handlers
	r.HandleFunc("/", renderTemplateHandler("index.html"))
	r.HandleFunc("/login", renderTemplateHandler("login.html"))
	r.HandleFunc("/register", renderTemplateHandler("register.html"))

	// Login/register post handlers
	u.HandleFunc(registrationHandler.Path, registrationHandler.Register)
	u.HandleFunc(loginHandler.Path, loginHandler.Login)

	// Authenticated pages
	///////////////////////////////
	auth.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		tmplPath := filepath.Join("tmpl", "feed.html")
		tmpl := template.Must(template.ParseFiles(tmplPath, tmplBasePath))

		user, _ := parseJwt(r)

		usersPosts := repo.GetFollowedPosts(user)
		for i, post := range usersPosts {
			usersPosts[i].Likes = repo.GetLikes(post.Id)
		}
		users := repo.GetUsers(user)

		tmpl.ExecuteTemplate(w, "base", map[string]interface{}{"You": user, "Others": users, "Posts": usersPosts})
	})

	auth.HandleFunc("/new_post", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			panic(err)
		}
		j, _ := parseJwt(r)

		post := post.Post{
			Body:   r.Form.Get("body"),
			Author: j,
		}

		repo.AddPost(&post)
		http.Redirect(w, r, "/auth/feed", http.StatusFound)
	})

	auth.HandleFunc("/id/{id}/like", func(w http.ResponseWriter, r *http.Request) {
		user, _ := parseJwt(r)
		id, _ := strconv.Atoi(mux.Vars(r)["id"])

		repo.LikePost(user, int64(id))
		http.Redirect(w, r, "/auth/feed", http.StatusFound)
	})

	auth.HandleFunc("/{user}/follow", func(w http.ResponseWriter, r *http.Request) {
		user, _ := parseJwt(r)
		follow := mux.Vars(r)["user"]
		log.Printf(follow)

		repo.FollowUser(user, follow)
		http.Redirect(w, r, "/auth/feed", http.StatusFound)
	})

	auth.HandleFunc("/{user}", func(w http.ResponseWriter, r *http.Request) {
		user := mux.Vars(r)["user"]

		tmplPath := filepath.Join("tmpl", "feed.html")
		tmpl := template.Must(template.ParseFiles(tmplPath, tmplBasePath))

		me, _ := parseJwt(r)

		usersPosts := repo.GetUserPosts(user)
		for i, post := range usersPosts {
			usersPosts[i].Likes = repo.GetLikes(post.Id)
		}
		users := repo.GetUsers(me)

		tmpl.ExecuteTemplate(w, "base", map[string]interface{}{"You": me, "Others": users, "Posts": usersPosts})
	})

	if err := http.ListenAndServe(os.Getenv("SWITTER_PORT"), r); err != nil {
		panic(err)
	}
}
