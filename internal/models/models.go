package models

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name" binding:"required"`
	Age   int    `json:"age" binding:"required,gt=0"`
	Email string `json:"email" binding:"omitempty,email,max=254"`
	City  string `json:"city" binding:"omitempty,max=120"`
}
type UserUpdate struct {
	Name  *string `json:"name"`
	Age   *int    `json:"age"`
	Email *string `json:"email" binding:"omitempty,email,max=254"`
	City  *string `json:"city" binding:"omitempty,max=120"`
}
