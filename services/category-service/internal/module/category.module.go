package module

import (
	"database/sql"

	"github.com/fairuzald/library-system/pkg/cache"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/proto/category"
	"github.com/fairuzald/library-system/services/category-service/internal/handler"
	"github.com/fairuzald/library-system/services/category-service/internal/repository"
	"github.com/fairuzald/library-system/services/category-service/internal/service"
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

	CategoryRepo    repository.CategoryRepository
	CategoryService service.CategoryService

	CategoryHandler     *handler.CategoryHandler
	HealthHandler       *handler.HealthHandler
	CategoryGRPCHandler *handler.CategoryGRPCHandler

	Log *logger.Logger
}

func New(
	db *sql.DB,
	redis *cache.Redis,
	jwtSecret string,
	log *logger.Logger,
) (*Module, error) {
	m := &Module{
		DB:    db,
		Redis: redis,
		Log:   log,
	}

	var err error
	m.GormDB, err = gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		log.Error("Failed to create GORM DB instance", zap.Error(err))
		return nil, err
	}

	m.JWTAuth = middleware.NewJWTAuth(jwtSecret, 0) // JWT duration not needed for this service

	m.CategoryRepo = repository.NewCategoryRepository(m.GormDB, redis, log)
	m.CategoryService = service.NewCategoryService(m.CategoryRepo, log)

	m.CategoryHandler = handler.NewCategoryHandler(m.CategoryService, log)
	m.HealthHandler = handler.NewHealthHandler(db, log)
	m.CategoryGRPCHandler = handler.NewCategoryGRPCHandler(m.CategoryService, log)

	return m, nil
}

func (m *Module) RegisterGRPCHandlers(grpcServer *grpc.Server) {
	category.RegisterCategoryServiceServer(grpcServer, m.CategoryGRPCHandler)
}
