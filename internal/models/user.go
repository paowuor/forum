package models

import "time"

type User struct {
	ID	         int64
	UUID         string
	Email        string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}