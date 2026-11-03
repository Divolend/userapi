//go:build integration

package handler

import (
	"os"
	"rest-api-gin/internal/database"
	"rest-api-gin/internal/models"
	"rest-api-gin/internal/repository"
	"strconv"
	"testing"
)

// CI supplies a disposable PostgreSQL database, never a deployed environment.
func TestPostgresIntegration(t *testing.T) {
	if os.Getenv("USERAPI_TEST_DATABASE") != "1" || os.Getenv("DB_NAME") != "userapi_lab7_test" {
		t.Fatal("integration tests require an explicitly isolated userapi_lab7_test database")
	}
	db, err := database.NewConnection()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	repo := repository.NewPostgresUserRepository(db)
	r := testRouter(t, repo)
	t.Run("HTTP CRUD persisted in PostgreSQL", func(t *testing.T) {
		request(t, r, "POST", "/users", `{"name":"IntegrationUser","age":21}`, "test", 200, "IntegrationUser")
		u, exists, err := repo.FindByName("IntegrationUser")
		if err != nil || !exists || u.Age != 21 {
			t.Fatalf("database user=%+v exists=%v err=%v", u, exists, err)
		}
		path := "/users/" + strconv.Itoa(u.ID)
		request(t, r, "GET", path, "", "", 200, "IntegrationUser")
		request(t, r, "POST", "/users", `{"name":"IntegrationUser","age":25}`, "test", 409, "")
		request(t, r, "PATCH", path, `{"name":"RenamedUser","age":24}`, "test", 200, "RenamedUser")
		saved, found, err := repo.FindByID(u.ID)
		if err != nil || !found || saved.Name != "RenamedUser" || saved.Age != 24 {
			t.Fatalf("update not persisted: %+v %v %v", saved, found, err)
		}
		request(t, r, "DELETE", path, "", "test", 200, "RenamedUser")
		request(t, r, "GET", path, "", "", 404, "")
		_, found, err = repo.FindByID(u.ID)
		if err != nil || found {
			t.Fatalf("deleted row still exists: %v %v", found, err)
		}
	})
	t.Run("database constraints", func(t *testing.T) {
		if _, err := repo.Create(models.User{Name: "ConstraintUser", Age: 21}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Create(models.User{Name: "ConstraintUser", Age: 22}); err == nil {
			t.Fatal("UNIQUE constraint allowed duplicate")
		}
		if _, err := repo.Create(models.User{Name: "InvalidAge", Age: 0}); err == nil {
			t.Fatal("CHECK constraint allowed age=0")
		}
	})
	t.Run("authorization prevents database write", func(t *testing.T) {
		request(t, r, "POST", "/users", `{"name":"UnauthorizedUser","age":21}`, "", 401, "")
		_, exists, err := repo.FindByName("UnauthorizedUser")
		if err != nil || exists {
			t.Fatalf("unauthorized insert: %v %v", exists, err)
		}
	})
	t.Run("email persisted and validated", func(t *testing.T) {
		request(t, r, "POST", "/users", `{"name":"EmailUser","age":25,"email":"diana@example.test"}`, "test", 200, "diana@example.test")
		u, found, err := repo.FindByName("EmailUser")
		if err != nil || !found || u.Email != "diana@example.test" {
			t.Fatalf("email not persisted: %+v %v", u, err)
		}
		path := "/users/" + strconv.Itoa(u.ID)
		request(t, r, "GET", path, "", "", 200, "diana@example.test")
		request(t, r, "PATCH", path, `{"email":"new@example.test"}`, "test", 200, "new@example.test")
		request(t, r, "POST", "/users", `{"name":"BadEmail","age":25,"email":"not-an-email"}`, "test", 400, "")
		request(t, r, "PATCH", path, `{"email":"invalid"}`, "test", 400, "")
		saved, _, err := repo.FindByID(u.ID)
		if err != nil || saved.Email != "new@example.test" {
			t.Fatalf("email changed after rejected patch: %+v %v", saved, err)
		}
	})
	t.Run("city persisted after migration", func(t *testing.T) {
		request(t, r, "POST", "/users", `{"name":"CityUser","age":24,"city":"Moscow"}`, "test", 200, "Moscow")
		u, found, err := repo.FindByName("CityUser")
		if err != nil || !found || u.City != "Moscow" {
			t.Fatalf("city not persisted: %+v %v", u, err)
		}
		path := "/users/" + strconv.Itoa(u.ID)
		request(t, r, "PATCH", path, `{"city":"Kazan"}`, "test", 200, "Kazan")
		request(t, r, "GET", path, "", "", 200, "Kazan")
		saved, _, err := repo.FindByID(u.ID)
		if err != nil || saved.City != "Kazan" {
			t.Fatalf("city update not persisted: %+v %v", saved, err)
		}
	})
}
