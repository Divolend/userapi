package service

import (
	"errors"
	"rest-api-gin/internal/models"
	"rest-api-gin/internal/repository"
	"testing"
)

// Embedding makes unexpected repository calls fail instead of silently succeeding.
type repositoryStub struct {
	repository.UserRepository
	findName func(string) (models.User, bool, error)
	findID   func(int) (models.User, bool, error)
	create   func(models.User) (models.User, error)
	update   func(int, models.UserUpdate) (models.User, error)
	delete   func(int) (models.User, error)
}

func (r repositoryStub) FindByName(n string) (models.User, bool, error) { return r.findName(n) }
func (r repositoryStub) FindByID(id int) (models.User, bool, error)     { return r.findID(id) }
func (r repositoryStub) Create(u models.User) (models.User, error)      { return r.create(u) }
func (r repositoryStub) UpdateField(id int, u models.UserUpdate) (models.User, error) {
	return r.update(id, u)
}
func (r repositoryStub) DeleteUser(id int) (models.User, error) { return r.delete(id) }

func TestCreateUser(t *testing.T) {
	user := models.User{ID: 1, Name: "Diana", Age: 21}
	dbErr := errors.New("storage unavailable")
	for _, tc := range []struct {
		name                 string
		exists               bool
		lookupErr, errorWant error
	}{
		{name: "new user"}, {name: "duplicate name", exists: true, errorWant: ErrNameAlreadyExists},
		{name: "lookup failure", lookupErr: dbErr, errorWant: dbErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			created := false
			r := repositoryStub{findName: func(n string) (models.User, bool, error) {
				if n != user.Name {
					t.Fatalf("lookup name=%q", n)
				}
				return user, tc.exists, tc.lookupErr
			}, create: func(u models.User) (models.User, error) { created = true; return u, nil }}
			got, err := NewUserService(r).CreateUser(user)
			if tc.exists {
				if err == nil {
					t.Fatal("duplicate was accepted")
				}
				if created {
					t.Fatal("duplicate reached Create")
				}
				return
			}
			if !errors.Is(err, tc.errorWant) {
				t.Fatalf("error=%v, want %v", err, tc.errorWant)
			}
			if err == nil && (got != user || !created) {
				t.Fatalf("created=%v user=%+v", created, got)
			}
		})
	}
	t.Run("insert failure", func(t *testing.T) {
		r := repositoryStub{findName: func(string) (models.User, bool, error) { return models.User{}, false, nil }, create: func(models.User) (models.User, error) { return models.User{}, dbErr }}
		_, err := NewUserService(r).CreateUser(user)
		if !errors.Is(err, dbErr) {
			t.Fatalf("error=%v", err)
		}
	})
}

func TestFindByID(t *testing.T) {
	dbErr := errors.New("storage unavailable")
	user := models.User{ID: 7, Name: "Diana", Age: 21}
	for _, tc := range []struct {
		name             string
		exists           bool
		repoErr, wantErr error
	}{
		{name: "found", exists: true}, {name: "missing", wantErr: repository.ErrUserNotFound}, {name: "storage failure", repoErr: dbErr, wantErr: dbErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := repositoryStub{findID: func(id int) (models.User, bool, error) {
				if id != 7 {
					t.Fatalf("id=%d", id)
				}
				return user, tc.exists, tc.repoErr
			}}
			got, err := NewUserService(r).FindByID(7)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error=%v want=%v", err, tc.wantErr)
			}
			if err == nil && got != user {
				t.Fatalf("user=%+v", got)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	current := models.User{ID: 7, Name: "Diana", Age: 21}
	renamed := "Yaroslava"
	same := "Diana"
	positive := 22
	zero := 0
	negative := -1
	for _, tc := range []struct {
		name               string
		fields             models.UserUpdate
		duplicate, missing bool
		wantErr            error
	}{
		{name: "positive age", fields: models.UserUpdate{Age: &positive}},
		{name: "zero age rejected", fields: models.UserUpdate{Age: &zero}, wantErr: ErrInvalidAge},
		{name: "negative age rejected", fields: models.UserUpdate{Age: &negative}, wantErr: ErrInvalidAge},
		{name: "rename", fields: models.UserUpdate{Name: &renamed}},
		{name: "same name allowed", fields: models.UserUpdate{Name: &same}},
		{name: "duplicate rejected", fields: models.UserUpdate{Name: &renamed}, duplicate: true, wantErr: ErrNameAlreadyExists},
		{name: "missing user", fields: models.UserUpdate{Name: &renamed}, missing: true, wantErr: repository.ErrUserNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			updated := false
			r := repositoryStub{findID: func(int) (models.User, bool, error) { return current, !tc.missing, nil }, findName: func(string) (models.User, bool, error) { return models.User{}, tc.duplicate, nil }, update: func(id int, f models.UserUpdate) (models.User, error) {
				updated = true
				if id != 7 {
					t.Fatalf("id=%d", id)
				}
				u := current
				if f.Name != nil {
					u.Name = *f.Name
				}
				if f.Age != nil {
					u.Age = *f.Age
				}
				return u, nil
			}}
			got, err := NewUserService(r).UpdateUser(7, tc.fields)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error=%v want=%v", err, tc.wantErr)
			}
			if tc.wantErr != nil && updated {
				t.Fatal("invalid update reached repository")
			}
			if err == nil {
				if !updated {
					t.Fatal("update was not saved")
				}
				if tc.fields.Age != nil && got.Age != *tc.fields.Age {
					t.Fatalf("age=%d", got.Age)
				}
				if tc.fields.Name != nil && got.Name != *tc.fields.Name {
					t.Fatalf("name=%s", got.Name)
				}
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	for _, tc := range []struct {
		name    string
		repoErr error
	}{{name: "existing"}, {name: "missing", repoErr: repository.ErrUserNotFound}} {
		t.Run(tc.name, func(t *testing.T) {
			user := models.User{ID: 7, Name: "Diana", Age: 21}
			r := repositoryStub{delete: func(id int) (models.User, error) {
				if id != 7 {
					t.Fatalf("id=%d", id)
				}
				return user, tc.repoErr
			}}
			got, err := NewUserService(r).DeleteUser(7)
			if !errors.Is(err, tc.repoErr) {
				t.Fatalf("error=%v", err)
			}
			if err == nil && got != user {
				t.Fatalf("user=%+v", got)
			}
		})
	}
}
