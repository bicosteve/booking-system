package repo

import (
	"database/sql"

	"github.com/redis/go-redis/v9"
)

type Repository struct {
	db    *sql.DB
	cache *redis.Client
}

func NewDBRepository(db *sql.DB, ch *redis.Client) *Repository {
	return &Repository{db: db, cache: ch}
}
