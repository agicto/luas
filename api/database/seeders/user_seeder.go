package seeders

import (
	"gorm.io/gorm"

	"github.com/zgiai/luas/api/internal/modules/user"
)

type UserSeeder struct{}

func (s *UserSeeder) Name() string {
	return "users"
}

func (s *UserSeeder) Run(db *gorm.DB) error {
	users := []user.UserPO{
		{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "$2a$10$OkAgF/Pm/v3pdzkUhKJEeOhehkbTRZar9Rk3X2nEjCcrlsluiTnay", // password: secret
			Nickname: "Administrator",
			Status:   1,
		},
		{
			Username: "user",
			Email:    "user@example.com",
			Password: "$2a$10$OkAgF/Pm/v3pdzkUhKJEeOhehkbTRZar9Rk3X2nEjCcrlsluiTnay", // password: secret
			Nickname: "Regular User",
			Status:   1,
		},
	}

	for _, u := range users {
		if err := db.FirstOrCreate(&u, user.UserPO{Email: u.Email}).Error; err != nil {
			return err
		}
	}

	return nil
}

func init() {
	register(&UserSeeder{})
}
