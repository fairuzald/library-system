package module

import (
	"database/sql"

	"github.com/fairuzald/library-system/pkg/cache"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/proto/book"
	"github.com/fairuzald/library-system/services/book-service/internal/handler"
	"github.com/fairuzald/library-system/services/book-service/internal/repository"
	"github.com/fairuzald/library-system/services/book-service/internal/service"
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

	CategoryClient service.CategoryClient
	BookRepo       repository.BookRepository
	BookService    service.BookService

	BookHandler     *handler.BookHandler
	HealthHandler   *handler.HealthHandler
	BookGRPCHandler *handler.BookGRPCHandler

	Log *logger.Logger
}

func New(
	db *sql.DB,
	redis *cache.Redis,
	jwtSecret string,
	categoryServiceURL string,
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

	m.CategoryClient, err = service.NewCategoryClient(categoryServiceURL, log)
	if err != nil {
		log.Warn("Failed to create category client, using mock client", zap.Error(err))
	}

	m.BookRepo = repository.NewBookRepository(m.GormDB, redis, log)
	m.BookService = service.NewBookService(m.BookRepo, m.CategoryClient, log)

	m.BookHandler = handler.NewBookHandler(m.BookService, log)
	m.HealthHandler = handler.NewHealthHandler(db, log)
	m.BookGRPCHandler = handler.NewBookGRPCHandler(m.BookService, log)

	return m, nil
}

func (m *Module) RegisterGRPCHandlers(grpcServer *grpc.Server) {
	book.RegisterBookServiceServer(grpcServer, m.BookGRPCHandler)
}

func (m *Module) Close() error {
	var err error
	if m.CategoryClient != nil {
		err = m.CategoryClient.Close()
	}
	return err
}
