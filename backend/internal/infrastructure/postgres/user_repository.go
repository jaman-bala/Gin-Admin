package postgres

import (
	"context"
	"database/sql"
	stdErrors "errors"
	"fmt"
	"gin_auth_service/internal/domain/user"
	"gin_auth_service/pkg/errors"
	"time"

	"uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint error.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return stdErrors.As(err, &pqErr) && pqErr.Code == "23505"
}

// UserDB represents the user record in the database.
type UserDB struct {
	ID         uuid.UUID  `db:"id"`
	FirstName  string     `db:"first_name"`
	LastName   string     `db:"last_name"`
	MiddleName string     `db:"middle_name"`
	Phone      string     `db:"phone"`
	Password   string     `db:"password"`
	Role       string     `db:"role"`
	Photo      string     `db:"photo"`
	Telegram   string     `db:"telegram"`
	IsActive   bool       `db:"is_active"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

func (m *UserDB) ToEntity() *user.User {
	return &user.User{
		ID:         m.ID,
		FirstName:  m.FirstName,
		LastName:   m.LastName,
		MiddleName: m.MiddleName,
		Phone:      m.Phone,
		Password:   m.Password,
		Role:       user.Role(m.Role),
		Photo:      m.Photo,
		Telegram:   m.Telegram,
		IsActive:   m.IsActive,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
		DeletedAt:  m.DeletedAt,
	}
}

func fromUserEntity(e *user.User) *UserDB {
	return &UserDB{
		ID:         e.ID,
		FirstName:  e.FirstName,
		LastName:   e.LastName,
		MiddleName: e.MiddleName,
		Phone:      e.Phone,
		Password:   e.Password,
		Role:       string(e.Role),
		Photo:      e.Photo,
		Telegram:   e.Telegram,
		IsActive:   e.IsActive,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
		DeletedAt:  e.DeletedAt,
	}
}

type userRepository struct {
	base
}

// NewUserRepository creates a new instance of UserRepository.
func NewUserRepository(db *sqlx.DB) user.Repository {
	return &userRepository{base{db: db}}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (id, first_name, last_name, middle_name, phone, password, role, photo, telegram, is_active, created_at, updated_at)
		VALUES (:id, :first_name, :last_name, :middle_name, :phone, :password, :role, :photo, :telegram, :is_active, :created_at, :updated_at)
	`
	dbModel := fromUserEntity(u)
	_, err := sqlx.NamedExecContext(ctx, r.q(ctx), query, dbModel)
	if err != nil {
		if isUniqueViolation(err) {
			return errors.ErrConflict
		}
		return err
	}
	return nil
}

func (r *userRepository) GetWithFilters(ctx context.Context, params user.FilterParams) ([]*user.User, int, error) {
	// Build dynamic WHERE clause
	where := " WHERE deleted_at IS NULL"
	args := []any{}
	argIdx := 1

	if params.Search != "" {
		where += ` AND (
			first_name ILIKE $` + fmt.Sprintf("%d", argIdx) + ` OR
			last_name  ILIKE $` + fmt.Sprintf("%d", argIdx) + ` OR
			phone      ILIKE $` + fmt.Sprintf("%d", argIdx) + `
		)`
		args = append(args, "%"+params.Search+"%")
		argIdx++
	}

	if params.IsActive != nil {
		where += ` AND is_active = $` + fmt.Sprintf("%d", argIdx)
		args = append(args, *params.IsActive)
	}

	usersDB, total, err := r.selectPage[UserDB](ctx, "users", where, "created_at DESC", params.Page, params.Limit, args...)
	if err != nil {
		return nil, 0, err
	}
	return mapSlice(usersDB, (*UserDB).ToEntity), int(total), nil
}

func (r *userRepository) GetID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var uDB UserDB
	query := `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`
	err := sqlx.GetContext(ctx, r.q(ctx), &uDB, query, id)
	if err != nil {
		if stdErrors.Is(err, sql.ErrNoRows) {
			return nil, errors.ErrUserNotFound
		}
		return nil, err
	}
	return uDB.ToEntity(), nil
}

func (r *userRepository) Patch(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET first_name=:first_name,
			last_name=:last_name,
			middle_name=:middle_name,
			phone=:phone,
			password=:password,
			role=:role,
			photo=:photo,
			telegram=:telegram,
			is_active=:is_active,
			updated_at=:updated_at
		WHERE id=:id AND deleted_at IS NULL
	`
	dbModel := fromUserEntity(u)
	_, err := sqlx.NamedExecContext(ctx, r.q(ctx), query, dbModel)
	if err != nil {
		if isUniqueViolation(err) {
			return errors.ErrPhoneAlreadyExists
		}
		return err
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1`
	_, err := r.q(ctx).ExecContext(ctx, query, id)
	return err
}

func (r *userRepository) FindByPhone(ctx context.Context, phone string) (*user.User, error) {
	var uDB UserDB
	query := `SELECT * FROM users WHERE phone = $1 AND deleted_at IS NULL`
	err := sqlx.GetContext(ctx, r.q(ctx), &uDB, query, phone)
	if err != nil {
		if stdErrors.Is(err, sql.ErrNoRows) {
			return nil, errors.ErrUserNotFound
		}
		return nil, err
	}
	return uDB.ToEntity(), nil
}
