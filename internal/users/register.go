package users

import (
	"net/http"
)

type UserRegistration struct {
	User User `json:"user"`
}

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}

type UserRegistrationHandler struct {
	Path           string
	UserRepository UserRepository
}

func (u *UserRegistrationHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		panic(err)
	}
	user := User{
		Username: r.Form.Get("username"),
		Email:    r.Form.Get("email"),
		Password: r.Form.Get("password"),
	}

	if user.Username == "" || user.Email == "" || user.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_ = u.UserRepository.RegisterUser(&user)
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}
