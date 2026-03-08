package domain

import "time"

type User struct {
	ID        string     `json:"id" bson:"_id"`
	Username  string     `json:"username" bson:"username"`
	Balance   float64    `json:"balance" bson:"balance"`
	Status    UserStatus `json:"status" bson:"status"`
	CreatedAt time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" bson:"updated_at"`
}
