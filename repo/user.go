package repo

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
)

// 1. UserRepository interface defines methods to interact with user data
type UserRepository interface {
	CreateUser(user entities.UserPayload) error
	FindUserByEmail(user entities.UserPayload) error
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
func (r *UserDBRepository) CreateUser(ctx context.Context, user entities.UserPayload) error {
	requestID := ctx.Value("request_id")
	q := `INSERT INTO 
			user(email,phone_number,isVender,hashed_password, created_at, 
			updated_at) VALUES (?,?,?,?,NOW(),NOW())`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		slog.Error("%s failed to prepare because of %s", "error", requestID, slog.String(err.Error(), "register"))
		return err
	}

	defer stmt.Close()

	hash, err := utils.GeneratePasswordHash(user.Password)
	if err != nil {
		slog.Error("failed to insert user because of %v", "error", err)
		return err

	}

	_, err = stmt.ExecContext(ctx, user.Email, user.PhoneNumber, user.IsVendor, hash)
	if err != nil {
		slog.Error("failed to insert user because of %v", "error", err)
		return err
	}

	return nil

}

func (r *UserDBRepository) FindUserByEmail(ctx context.Context, email string) (bool, error) {

	var count int

	q := `SELECT COUNT(*) FROM user WHERE email = ?`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		slog.Error("failed to prepare the statement due to %v", "error", err)
		return false, err
	}

	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, email)

	err = row.Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, sql.ErrNoRows
		}
		slog.Error("failed because of %v ", "error", err)
		return false, err
	}

	return count > 0, nil
}

func (r *UserDBRepository) FindAProfile(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User

	q := `SELECT * FROM user WHERE email = ?`
	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, email)

	err = row.Scan(&user.ID, &user.Email, &user.PhoneNumber, &user.IsVendor, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		slog.Error("failed to execute statement due to %v ", "error", err)
		return nil, err
	}

	return &user, nil
}
