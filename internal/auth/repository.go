package auth

import (
	"context"
	"os"

	rkpostgres "github.com/rookie-ninja/rk-db/postgres"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID           int64  `gorm:"primary_key"`
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
}
type AuthRepository struct {
	Db *gorm.DB
}

func NewAuthRepository() *AuthRepository {
	pgEntry := rkpostgres.GetPostgresEntry("auth-conn")
	dbName := os.Getenv("AUTH_DB_NAME")
	db := pgEntry.GetDB(dbName)
	db.AutoMigrate(&User{})
	return &AuthRepository{Db: db}
}

func (r *AuthRepository) CreateUser(ctx context.Context, email, password string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &User{
		Email:        email,
		PasswordHash: string(hashedPassword),
	}
	res := r.Db.WithContext(ctx).Create(user)
	return user, res.Error
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	res := r.Db.WithContext(ctx).First(&user, "email = ?", email)
	return &user, res.Error
}
