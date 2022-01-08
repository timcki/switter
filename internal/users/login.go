package users

import (
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type UserLogin struct {
	User User `json:"user"`
}

type LoggedInUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Token    string `json:"token"`
}

type UserLoginHandler struct {
	Path           string
	UserRepository UserRepository
}

type CustomClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func (u *UserLoginHandler) Login(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		panic(err)
	}

	requestUser := User{
		Username: r.Form.Get("username"),
		Password: r.Form.Get("password"),
	}

	if found, _ := u.UserRepository.FindByUserAndPassword(requestUser.Username, requestUser.Password); !found {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//token, _ := CreateToken(requestUser.Username)

	http.SetCookie(w, &http.Cookie{
		Name:    "jwt_token",
		Value:   requestUser.Username,
		Expires: time.Now().Add(time.Hour * 24),
		Path:    "/",
	})
	http.Redirect(w, r, "/auth/feed", http.StatusFound)
}

func CreateToken(username string) (string, error) {
	claims := CustomClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: 15000,
			Issuer:    "timchmielecki.com",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(os.Getenv("JWT_KEY")))
}
