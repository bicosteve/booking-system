package repo

import (
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewDBRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}
