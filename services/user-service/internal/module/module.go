package module

import (
	"context"
	"database/sql"
	"time"

	"github.com/fairuzald/library-system/pkg/cache"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/proto/user"
	"github.com/fairuzald/library-system/services/user-service/internal/handler"
	grpcHandler "github.com/fairuzald/library-system/services/user-service/internal/handler"
	"github.com/fairuzald/library-system/services/user-service/internal/repository"
	"github.com/fairuzald/library-system/services/user-service/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Module struct {
	DB     *sql.DB
	GormDB *gorm.DB

	Redis *cache.Redis

	JWTAuth *middleware.JWTAuth

	UserRepo repository.UserRepository
	AuthRepo repository.AuthRepository

	UserService service.UserService
	AuthService service.AuthService

	UserHandler   *handler.UserHandler
	AuthHandler   *handler.AuthHandler
	HealthHandler *handler.HealthHandler

	UserGRPCService *grpcHandler.UserService

	Log                *logger.Logger
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

func New(
	db *sql.DB,
	redis *cache.Redis,
	jwtSecret string,
	accessTokenExpiry time.Duration,
	refreshTokenExpiry time.Duration,
	log *logger.Logger,
) (*Module, error) {
	m := &Module{
		DB:                 db,
		Redis:              redis,
		Log:                log,
		AccessTokenExpiry:  accessTokenExpiry,
		RefreshTokenExpiry: refreshTokenExpiry,
	}

	var err error
	m.GormDB, err = gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		log.Error("Failed to create GORM DB instance", zap.Error(err))
		return nil, err
	}

	m.JWTAuth = middleware.NewJWTAuth(jwtSecret, accessTokenExpiry)

	m.UserRepo = repository.NewUserRepository(m.GormDB, redis, log)
	m.AuthRepo = repository.NewAuthRepository(m.GormDB, redis, log)

	m.UserService = service.NewUserService(m.UserRepo, log)
	m.AuthService = service.NewAuthService(m.UserRepo, m.AuthRepo, m.JWTAuth, log, accessTokenExpiry, refreshTokenExpiry)

	m.UserHandler = handler.NewUserHandler(m.UserService, log)
	m.AuthHandler = handler.NewAuthHandler(m.AuthService, log)
	m.HealthHandler = handler.NewHealthHandler(db, log)

	m.UserGRPCService = grpcHandler.NewUserService(m.UserService, m.AuthService, log)

	return m, nil
}

func (m *Module) RegisterGRPCHandlers(grpcServer *grpc.Server) {
	user.RegisterUserServiceServer(grpcServer, m.UserGRPCService)
}

func (m *Module) StartBackgroundTasks() {
	go m.startTokenCleanupTask()
}

func (m *Module) startTokenCleanupTask() {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := m.AuthRepo.CleanupExpiredTokens(ctx); err != nil {
			m.Log.Error("Failed to cleanup expired tokens", zap.Error(err))
		}
		cancel()
	}
}
