package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"rest-api-gin/internal/models"
	"strings"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(user models.User) (models.User, error)
	FindByName(name string) (models.User, bool, error)
	FindByID(id int) (models.User, bool, error)
	UpdateField(id int, fields models.UserUpdate) (models.User, error)
	DeleteUser(id int) (models.User, error)
}
type InMemoryUserRepository struct {
	users  []models.User
	nextId int
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:  []models.User{},
		nextId: 1,
	}

}

func (r *InMemoryUserRepository) Create(user models.User) (models.User, error) {
	user.ID = r.nextId
	r.nextId++
	r.users = append(r.users, user)
	return user, nil

}

func (r *InMemoryUserRepository) FindByName(name string) (models.User, bool, error) {
	for _, u := range r.users {
		if u.Name == name {
			return u, true, nil

		}
	}
	return models.User{}, false, nil
}
func (r *InMemoryUserRepository) FindByID(id int) (models.User, bool, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, true, nil
		}
	}
	return models.User{}, false, nil
}
func (r *InMemoryUserRepository) UpdateField(id int, fields models.UserUpdate) (models.User, error) {

	for i := range r.users {
		if r.users[i].ID == id {
			if fields.Age != nil {
				r.users[i].Age = *fields.Age
			}
			if fields.Name != nil {
				r.users[i].Name = *fields.Name
			}
			return r.users[i], nil

		}
	}
	return models.User{}, ErrUserNotFound

}

func (r *InMemoryUserRepository) DeleteUser(id int) (models.User, error) {
	for i := range r.users {
		if r.users[i].ID == id {
			deleteUser := r.users[i]
			r.users = append(r.users[:i], r.users[(i+1):]...)

			return deleteUser, nil

		}

	}
	return models.User{}, ErrUserNotFound

}


type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(DB *sql.DB) *PostgresUserRepository {

	return &PostgresUserRepository{db: DB}

}

func (r *PostgresUserRepository) Create(user models.User) (models.User, error) {

	err := r.db.QueryRow("INSERT INTO users(name,age) values($1,$2) RETURNING id", user.Name, user.Age).Scan(&user.ID)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to create user : %w", err)

	}
	return user, nil

}

func (r *PostgresUserRepository) FindByName(name string) (models.User, bool, error) {
	var u models.User
	err := r.db.QueryRow("SELECT * FROM users WHERE name = $1", name).Scan(&u.ID, &u.Name, &u.Age)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, false, nil
		}
		return models.User{}, false, fmt.Errorf("failed to select by name: %w", err)
	}

	return u, true, nil

}
func (r *PostgresUserRepository) FindByID(id int) (models.User, bool, error) {
	var u models.User
	err := r.db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(&u.ID, &u.Name, &u.Age)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, false, nil
		}
		return models.User{}, false, fmt.Errorf("failed to select by id: %w", err)
	}
	return u, true, nil
}
func (r *PostgresUserRepository) UpdateField(id int, fields models.UserUpdate) (models.User, error) {
	setParts := []string{}
	args := []any{}
	argIndex := 1
	if fields.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *fields.Name)
		argIndex++

	}
	if fields.Age != nil {
		setParts = append(setParts, fmt.Sprintf("age = $%d", argIndex))
		args = append(args, *fields.Age)
		argIndex++

	}
	query := "UPDATE users SET " + strings.Join(setParts, ", ") + fmt.Sprintf(" WHERE id = $%d RETURNING id,name,age", argIndex)
	args = append(args, id)
	var u models.User
	err := r.db.QueryRow(query, args...).Scan(&u.ID, &u.Name, &u.Age)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrUserNotFound

		}
		return models.User{}, fmt.Errorf("failed to update user : %w", err)

	}
	return u, nil

}

func (r *PostgresUserRepository) DeleteUser(id int) (models.User, error) {
	var u models.User

	err := r.db.QueryRow("DELETE FROM users WHERE id = $1 RETURNING id, name, age", id).Scan(&u.ID, &u.Name, &u.Age)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrUserNotFound

		}
		return models.User{}, fmt.Errorf("failed to delete user : %w", err)
	}
	return u, nil

}