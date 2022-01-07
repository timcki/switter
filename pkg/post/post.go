package post

import "github.com/timcki/switter_neo4j/pkg/users"

type Post struct {
	Author users.User
	Title  string
}
