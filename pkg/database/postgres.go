package database

import (
	"context"
	"fmt"
	"time"

	"github.com/fairuzald/library-system/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type PostgresDB struct {
	*gorm.DB
	log *logger.Logger
}

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	Logger   *logger.Logger
}

func NewPostgresDB(cfg *Config) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	log := cfg.Logger
	if log == nil {
		log = logger.Default()
	}

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
		PrepareStmt: true,
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Info("Successfully connected to PostgreSQL database", zap.String("database", cfg.DBName))

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &PostgresDB{
		DB:  db,
		log: log,
	}, nil
}

func (p *PostgresDB) Close() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	return sqlDB.Close()
}

func (p *PostgresDB) Ping(ctx context.Context) error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

func (p *PostgresDB) AutoMigrate(models ...interface{}) error {
	if err := p.DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to auto migrate models: %w", err)
	}
	return nil
}

func (p *PostgresDB) WithTransaction(fn func(tx *gorm.DB) error) error {
	return p.DB.Transaction(fn)
}

func (p *PostgresDB) Get(ctx context.Context, model interface{}, id interface{}) error {
	tx := p.WithContext(ctx).First(model, "id = ?", id)
	return tx.Error
}

func (p *PostgresDB) Create(ctx context.Context, model interface{}) error {
	tx := p.WithContext(ctx).Create(model)
	return tx.Error
}

func (p *PostgresDB) Update(ctx context.Context, model interface{}) error {
	tx := p.WithContext(ctx).Save(model)
	return tx.Error
}

func (p *PostgresDB) Delete(ctx context.Context, model interface{}, id interface{}) error {
	tx := p.WithContext(ctx).Delete(model, "id = ?", id)
	return tx.Error
}
