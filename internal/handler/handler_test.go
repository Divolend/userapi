package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"rest-api-gin/internal/models"
	"rest-api-gin/internal/repository"
	"rest-api-gin/internal/service"
	"strings"
	"testing"
)

func testRouter(t *testing.T, repo repository.UserRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previous := UserService
	UserService = service.NewUserService(repo)
	t.Cleanup(func() { UserService = previous })
	r := gin.New()
	r.Use(LoggerMiddleware)
	r.GET("/users/:id", IdHandler)
	r.POST("/users", AuthMiddleware, CreateUserHandler)
	r.PATCH("/users/:id", AuthMiddleware, UpdateHandler)
	r.DELETE("/users/:id", AuthMiddleware, DeleteHandler)
	return r
}
func request(t *testing.T, r http.Handler, method, path, body, auth string, want int, contains string) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != want {
		t.Fatalf("%s %s: status=%d, want=%d, body=%s", method, path, w.Code, want, w.Body.String())
	}
	if contains != "" && !strings.Contains(w.Body.String(), contains) {
		t.Fatalf("response=%q, want substring=%q", w.Body.String(), contains)
	}
}
func TestAuthMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name, auth string
		want       int
	}{{"missing header", "", 401}, {"nonempty header", "lab7-test", 204}} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			called := false
			r.POST("/protected", AuthMiddleware, func(c *gin.Context) { called = true; c.Status(204) })
			request(t, r, "POST", "/protected", "", tc.auth, tc.want, "")
			if called != (tc.want == 204) {
				t.Fatalf("protected handler called=%v", called)
			}
		})
	}
}
func TestHTTPScenarios(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body, auth string
		want                           int
		contains                       string
	}{
		{"create", "POST", "/users", `{"name":"Nastya","age":22}`, "test", 200, "Nastya"},
		{"public read", "GET", "/users/1", "", "", 200, "Diana"},
		{"missing read", "GET", "/users/999", "", "", 404, ""},
		{"bad read ID", "GET", "/users/abc", "", "", 400, ""},
		{"create without auth", "POST", "/users", `{"name":"Nastya","age":22}`, "", 401, ""},
		{"duplicate name", "POST", "/users", `{"name":"Diana","age":22}`, "test", 409, ""},
		{"invalid JSON", "POST", "/users", `{`, "test", 400, ""},
		{"missing name", "POST", "/users", `{"age":22}`, "test", 400, ""},
		{"missing age", "POST", "/users", `{"name":"Nastya"}`, "test", 400, ""},
		{"zero age", "POST", "/users", `{"name":"Nastya","age":0}`, "test", 400, ""},
		{"negative age", "POST", "/users", `{"name":"Nastya","age":-1}`, "test", 400, ""},
		{"update age", "PATCH", "/users/1", `{"age":23}`, "test", 200, "23"},
		{"rename", "PATCH", "/users/1", `{"name":"Nastya"}`, "test", 200, "Nastya"},
		{"duplicate rename", "PATCH", "/users/1", `{"name":"Yaroslava"}`, "test", 409, ""},
		{"invalid update age", "PATCH", "/users/1", `{"age":-1}`, "test", 400, ""},
		{"missing update", "PATCH", "/users/999", `{"age":23}`, "test", 404, ""},
		{"update without auth", "PATCH", "/users/1", `{"age":23}`, "", 401, ""},
		{"invalid update JSON", "PATCH", "/users/1", `{`, "test", 400, ""},
		{"bad update ID", "PATCH", "/users/abc", `{"age":23}`, "test", 400, ""},
		{"delete", "DELETE", "/users/1", "", "test", 200, "Diana"},
		{"missing delete", "DELETE", "/users/999", "", "test", 404, ""},
		{"delete without auth", "DELETE", "/users/1", "", "", 401, ""},
		{"bad delete ID", "DELETE", "/users/abc", "", "test", 400, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := repository.NewInMemoryUserRepository()
			for _, u := range []models.User{{Name: "Diana", Age: 21}, {Name: "Yaroslava", Age: 20}} {
				if _, err := repo.Create(u); err != nil {
					t.Fatal(err)
				}
			}
			r := testRouter(t, repo)
			request(t, r, tc.method, tc.path, tc.body, tc.auth, tc.want, tc.contains)
			if tc.want >= 400 {
				u, exists, err := repo.FindByID(1)
				if err != nil || !exists || u.Name != "Diana" || u.Age != 21 {
					t.Fatalf("rejected request changed original user: %+v %v %v", u, exists, err)
				}
			}
		})
	}
}
