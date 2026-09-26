package handler

import (
	"errors"
	"fmt"
	"net/http"
	"rest-api-gin/internal/models"
	"rest-api-gin/internal/repository"
	"rest-api-gin/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

var UserService *service.UserService

// createUserhandler godoc
// @Summary Создание нового пользователя
// @Tags users
// @Accept json
// @Produce plain
// @Param user body User true "Данные нового пользоавателя"
// @Success 200 {string} string "Пользователь успешно создан"
// @Failure      400  {string}  string  "Некорректный JSON в теле запроса"
// @Failure      409  {string}  string  "Пользователь с таким именем уже существует"
// @Router /users [post]
func CreateUserHandler(c *gin.Context) {
	var u models.User
	err := c.ShouldBindJSON(&u)
	if err != nil {
		fmt.Println(err)
		c.String(http.StatusBadRequest, "Bad Request\n")
		return
	}
	createdUser, err := UserService.CreateUser(u)
	if err != nil {
		c.String(http.StatusConflict, err.Error())
		return
	}

	c.String(200, "Данные нового пользователя: ID - %d Имя - %s, возраст - %d\n", createdUser.ID, createdUser.Name, createdUser.Age)
}


func IdHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "id - not an integer")
		return
	}
	user, err := UserService.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.String(http.StatusNotFound, err.Error())
			return

		}
		c.String(http.StatusBadRequest, err.Error())
		return

	}
	c.String(200, "По данному айди найден пользователь: ID - %d, Имя - %s, возраст - %d\n", user.ID, user.Name, user.Age)

}
func UpdateHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "id - not an integer")
		return
	}
	var fields models.UserUpdate
	err = c.ShouldBindJSON(&fields)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Request\n")
		return
	}
	updateUser, err := UserService.UpdateUser(id, fields)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.String(http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, service.ErrNameAlreadyExists) {
			c.String(http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidAge) {
			c.String(http.StatusBadRequest, err.Error())
			return

		}
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	c.String(http.StatusOK, fmt.Sprintf("Обновленные данные пользователя: Id : %d, имя : %s, возраст: %d\n", updateUser.ID, updateUser.Name, updateUser.Age))

}
func DeleteHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "id - not an integer")
		return

	}
	user, err := UserService.DeleteUser(id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.String(http.StatusNotFound, err.Error())
			return
		}
		c.String(http.StatusBadRequest, err.Error())
		return

	}

	c.String(http.StatusOK, "Пользователь : (ID - %d, имя - %s, возраст - %d) - успешно удален!", user.ID, user.Name, user.Age)

}
func LoggerMiddleware(c *gin.Context) {
	fmt.Println("Пришел запрос: ", c.Request.Method, c.Request.URL.Path)
}
func AuthMiddleware(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.String(http.StatusUnauthorized, "Unauthorized\n")
		c.Abort()
		return

	}
	c.Next()

}

