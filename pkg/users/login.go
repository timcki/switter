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

func (u *UserLoginHandler) Login(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		panic(err)
	}
	requestUser := User{
		Username: r.Form.Get("username"),
		Password: r.Form.Get("password"),
	}

	user, _ := u.UserRepository.FindByUserAndPassword(
		requestUser.Username,
		requestUser.Password)

	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token, _ := CreateToken(user)
	r.AddCookie(&http.Cookie{
		Name:  "token",
		Value: token,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func CreateToken(user *User) (string, error) {
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = user.Username
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString([]byte(os.Getenv("SECRET_ACCESS")))
}
