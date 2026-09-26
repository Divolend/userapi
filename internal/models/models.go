package models

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name" binding:"required"`
	Age  int    `json:"age" binding:"required,gt=0"`
}
type UserUpdate struct {
	Name *string `json:"name"`
	Age  *int    `json:"age"`
}
