package entity

import (
	"errors"
	"time"
)

type Book struct {
	ID        string
	Name      string
	AuthorIDs []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var ErrBookNotFound = errors.New("book not found")
