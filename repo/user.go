package repo

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/bicosteve/booking-system/entities"
)

// 1. UserRepository interface defines methods to interact with user data
type UserRepository interface {
	CreateUser(user entities.UserPayload) error
}

// 2. UserDBRepository implements UserRepository interface for mysql
type UserDBRepository struct {
	db *sql.DB
}

// 3. NewUserDBRepository creates a new instance of UserDBRepository
func NewUserDBRepository(db *sql.DB, ctx context.Context) *UserDBRepository {
	return &UserDBRepository{db: db}
}

// 4. Creates a user into the db
func (r *UserDBRepository) CreateUser(user entities.UserPayload) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	q := `INSERT INTO 
			user(email,phone_number,isVender,hashed_password, created_at, 
			updated_at) VALUES (?,?,?,?,NOW(),NOW())`

	_, err := r.db.ExecContext(ctx, q, user.Email, user.PhoneNumber, user.IsVendor, user.Password)
	if err != nil {
		slog.Error("failed to make connection because of %v", "error", err)
		return err
	}

	return nil

	// Getting a nil pointer dereferencing error here
	// conn, err := r.db.Conn(r.ctx)
	// if err != nil {
	// 	slog.Error("failed to make connection because of %v", "error", err)
	// 	return err
	// }

	// stmt, err := conn.PrepareContext(r.ctx, q)
	// if err != nil {
	// 	slog.Error("failed to prepare statement because of %v", "error", err)
	// 	return err
	// }

	// _, err = stmt.ExecContext(r.ctx, user.Email, user.PhoneNumber, user.IsVendor, user.Password, time.Now(), time.Now())
	// if err != nil {
	// 	slog.Error("failed to insert user because of %v", "error", err)
	// 	return err
	// }
}
