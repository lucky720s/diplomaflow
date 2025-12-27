//go:generate wire
//go:build wireinject
// +build wireinject

package auth

import (
	"time"

	"github.com/google/wire"
	"github.com/lucky720s/diplomaflow/pkg/database"
	"github.com/lucky720s/diplomaflow/pkg/logger"
	rolev1 "github.com/lucky720s/diplomaflow/pkg/protobuf/role/v1"
	universityv1 "github.com/lucky720s/diplomaflow/pkg/protobuf/university/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

func ProvideDB(cfg *Config) (*gorm.DB, func(), error) {
	return database.NewConnection(cfg.Database.DSN)
}

func ProvideUniversityClient(cfg *Config) (universityv1.UniversityServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.UniversityAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	client := universityv1.NewUniversityServiceClient(conn)
	cleanup := func() { conn.Close() }

	return client, cleanup, nil
}

func ProvideRoleClient(cfg *Config) (rolev1.RoleServiceClient, func(), error) {
	conn, err := grpc.NewClient(cfg.Services.RoleAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	client := rolev1.NewRoleServiceClient(conn)
	cleanup := func() { conn.Close() }

	return client, cleanup, nil
}

func ProvideJwtWrapper(cfg *Config) JwtWrapper {
	accessTTL, _ := time.ParseDuration(cfg.JWT.AccessTokenTTL)
	refreshTTL, _ := time.ParseDuration(cfg.JWT.RefreshTokenTTL)

	return JwtWrapper{
		SecretKey:       cfg.JWT.Secret,
		Issuer:          "diplomaflow",
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	}
}

func InitializeApp(cfg *Config, log *logger.Logger) (*Handler, func(), error) {
	wire.Build(
		ProvideDB,
		ProvideUniversityClient,
		ProvideRoleClient,
		ProvideJwtWrapper,
		NewRepository,
		NewService,
		wire.Bind(new(AuthService), new(*Service)),
		NewHandler,
	)
	return &Handler{}, nil, nil
}
