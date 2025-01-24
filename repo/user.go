package repo

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
)

// 1. UserRepository interface defines methods to interact with user data
type UserRepository interface {
	CreateUser(ctx context.Context, user entities.UserPayload) error
	FindUserByEmail(ctx context.Context, user entities.UserPayload) error
	UpdatePassword(ctx context.Context, user entities.UserPayload) error
	FindAProfile(ctx context.Context, email string) (*entities.User, error)
	InsertPasswordResetToken(ctx context.Context, resetToken string, userId int) error
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
	q := `
			INSERT INTO 
			user(email,phone_number,isVender,hashed_password, created_at, 
			updated_at, password_inserted_at) VALUES (?,?,?,?,NOW(),NOW(), NOW())
		`

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

	args := []interface{}{user.Email, user.PhoneNumber, user.IsVendor, hash}

	_, err = stmt.ExecContext(ctx, args...)
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

	err = row.Scan(&user.ID, &user.Email, &user.PhoneNumber, &user.IsVender, &user.Password, &user.PasswordResetToken, &user.CreatedAt, &user.UpdatedAt, &user.PasswordInsertedAt)
	if err != nil {
		slog.Error("failed to execute statement due to %v ", "error", err)
		return nil, err
	}

	return &user, nil
}

func (r *UserDBRepository) InsertPasswordResetToken(ctx context.Context, resetToken string, email string) error {
	q := `UPDATE user SET password_reset_token = ?, updated_at = ? WHERE email = ?`

	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	args := []interface{}{resetToken, time.Now(), email}

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *UserDBRepository) UpdatePassword(ctx context.Context, newPassword *string, userId int) error {

	q := `
		UPDATE user SET hash_password = ?, updated_at = ?, password_inserted_at = ? WHERE user_id = ?
	`
	stmt, err := r.db.PrepareContext(ctx, q)
	if err != nil {
		return err
	}

	defer stmt.Close()

	hash, err := utils.GeneratePasswordHash(*newPassword)
	if err != nil {
		return err
	}

	args := []interface{}{hash, time.Now(), time.Now(), userId}

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return err
	}

	return nil
}
