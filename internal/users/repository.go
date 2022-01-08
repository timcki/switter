package users

import (
	"errors"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j/dbtype"
	"github.com/timcki/switter/internal/post"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	RegisterUser(*User) error
	FindByUserAndPassword(string, string) (bool, error)
}

type UserNeo4jRepository struct {
	Driver neo4j.Driver
}

func (u *UserNeo4jRepository) GetUserPosts(username string) []post.Post {
	session := u.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	posts := make([]post.Post, 0)

	// Create post with relationship to author (no need for any keys this way)
	_, _ = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		createQuery := "MATCH (:User {username: $username})-[:POSTED]->(p) return p"
		params := map[string]interface{}{
			"username": username,
		}
		result, err := tx.Run(createQuery, params)
		if err != nil {
			return nil, err
		}
		records, err := result.Collect()
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			body := r.Values[0].(dbtype.Node).Props["body"].(string)
			posts = append(posts, post.Post{Author: username, Body: body})
		}
		return nil, nil
	})
	return posts
}

func (u *UserNeo4jRepository) GetUsers(me string) []string {
	session := u.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	users := make([]string, 0)

	// Create post with relationship to author (no need for any keys this way)
	_, _ = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := "match (u:User {username: $username})-[:FOLLOWS]->(o:User) with collect(o) as others match (w:User) where not w in others return w"
		params := map[string]interface{}{
			"username": me,
		}
		result, err := tx.Run(query, params)
		if err != nil {
			return nil, err
		}
		records, err := result.Collect()
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			user := r.Values[0].(dbtype.Node).Props["username"].(string)
			if user != me {
				users = append(users, user)
			}
		}
		return nil, nil
	})
	return users

}

func (u *UserNeo4jRepository) FollowUser(username, follow string) error {
	session := u.Driver.NewSession(neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		query := "MATCH (n:User {username: $username}),(m:User {username: $follow}) CREATE (n)-[:FOLLOWS]->(m) RETURN n, m"
		params := map[string]interface{}{
			"username": username,
			"follow":   follow,
		}
		return tx.Run(query, params)
	})
	return err
}

func (u *UserNeo4jRepository) GetFollowedPosts(username string) []post.Post {
	session := u.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	posts := make([]post.Post, 0)

	// Create post with relationship to author (no need for any keys this way)
	_, _ = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		createQuery := "MATCH (:User {username: $username})-[:FOLLOWS]->(f:User)-[:POSTED]->(p) return f, p"
		params := map[string]interface{}{
			"username": username,
		}
		result, err := tx.Run(createQuery, params)
		if err != nil {
			return nil, err
		}
		records, err := result.Collect()
		if err != nil {
			return nil, err
		}

		for _, r := range records {
			id := r.Values[1].(dbtype.Node).Id
			user := r.Values[0].(dbtype.Node).Props["username"].(string)
			body := r.Values[1].(dbtype.Node).Props["body"].(string)
			posts = append(posts, post.Post{Author: user, Body: body, Id: id})
		}
		return nil, nil
	})
	return posts
}

func (u *UserNeo4jRepository) LikePost(username string, id int64) (err error) {
	session := u.Driver.NewSession(neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close()

	// Create post with relationship to author (no need for any keys this way)
	_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		likeQuery := "MATCH (u:User {username: $username}), (p:Post) WHERE ID(p)=$id CREATE (u)-[:LIKES]->(p) return u, p"
		params := map[string]interface{}{
			"username": username,
			"id":       id,
		}
		return tx.Run(likeQuery, params)
	})
	return err
}

func (u *UserNeo4jRepository) AddPost(post *post.Post) (err error) {
	session := u.Driver.NewSession(neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close()

	// Create post with relationship to author (no need for any keys this way)
	_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		createQuery := "MATCH (u:User {username: $username}) CREATE (u)-[:POSTED]->(:Post {body: $body})"
		params := map[string]interface{}{
			"username": post.Author,
			"body":     post.Body,
		}
		return tx.Run(createQuery, params)
	})
	return err
}

func (u *UserNeo4jRepository) RegisterUser(user *User) (err error) {
	session := u.Driver.NewSession(neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})

	session.Close()

	if _, err := session.
		WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
			return u.persistUser(tx, user)
		}); err != nil {
		return err
	}
	return nil
}

func (u *UserNeo4jRepository) FindByUserAndPassword(username string, password string) (bool, error) {
	session := u.Driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	result, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		return u.findUser(tx, username, password)
	})
	if result == nil {
		return false, err
	}
	return result.(bool), err
}

func (u *UserNeo4jRepository) persistUser(tx neo4j.Transaction, user *User) (interface{}, error) {
	query := "CREATE (:User {email: $email, username: $username, password: $password})"
	parameters := map[string]interface{}{
		"email":    user.Email,
		"username": user.Username,
		"password": hash(user.Password),
	}

	return tx.Run(query, parameters)
}

func (u *UserNeo4jRepository) findUser(tx neo4j.Transaction, username string, password string) (bool, error) {
	var result neo4j.Result
	var err error

	if result, err = tx.Run(
		"MATCH (u:User {username: $username}) RETURN u.password AS password",
		map[string]interface{}{
			"username": username,
		},
	); err != nil {
		return false, err
	}

	if record, err := result.Single(); err != nil {
		return false, err
	} else {
		hashedPassword, _ := record.Get("password")
		if !passwordsMatch(hashedPassword.(string), password) {
			return false, errors.New("Password do not match")
		}
	}

	return true, nil
}

func passwordsMatch(hashedPassword string, clearTextPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(clearTextPassword)) == nil
}

func hash(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
}
