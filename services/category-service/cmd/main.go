package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fairuzald/library-system/pkg/cache"
	"github.com/fairuzald/library-system/pkg/config"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/services/category-service/internal/module"
	"github.com/fairuzald/library-system/services/category-service/internal/routes"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		panic(fmt.Sprintf("Error loading config: %v", err))
	}

	logConfig := config.LoadLoggingConfig()
	log := logger.New(logger.Config{
		Level:      logConfig.Level,
		Production: logConfig.Production,
		JsonFormat: logConfig.JsonFormat,
	})
	defer log.Sync()

	log.Info("Starting category service",
		zap.String("app_name", cfg.AppName),
		zap.String("env", cfg.AppEnv),
		zap.String("version", "1.0.0"),
	)

	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database", zap.Error(err))
	}
	log.Info("Successfully connected to database", zap.String("database", cfg.DBName))

	redisClient, err := cache.NewRedis(&cache.RedisConfig{
		Host:     cfg.RedisHost,
		Port:     cfg.RedisPort,
		Password: cfg.RedisPassword,
		DB:       0,
		Logger:   log,
	})
	if err != nil {
		log.Warn("Failed to connect to Redis, proceeding without cache", zap.Error(err))
	} else {
		log.Info("Successfully connected to Redis", zap.String("host", cfg.RedisHost))
		defer redisClient.Close()
	}

	// Create the module instance
	categoryModule, err := module.New(
		db,
		redisClient,
		cfg.JWTSecret,
		log,
	)
	if err != nil {
		log.Fatal("Failed to initialize category service module", zap.Error(err))
	}

	router := mux.NewRouter()

	requestLogger := middleware.NewRequestLogger(log)
	recoveryMiddleware := middleware.NewRecoveryMiddleware(log)
	router.Use(recoveryMiddleware.Middleware, requestLogger.Middleware)

	router.HandleFunc("/health", categoryModule.HealthHandler.HandleHealth).Methods("GET")

	// Set up routes
	routes.SetupRoutes(
		router,
		categoryModule.CategoryHandler,
		categoryModule.JWTAuth,
		log,
	)

	httpServer := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.UnaryRecoveryInterceptor(log),
			middleware.UnaryLoggingInterceptor(log),
		),
		grpc.ChainStreamInterceptor(
			middleware.StreamRecoveryInterceptor(log),
			middleware.StreamLoggingInterceptor(log),
		),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Minute,
			Timeout:               20 * time.Second,
		}),
	)

	// Register gRPC handlers
	categoryModule.RegisterGRPCHandlers(grpcServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("category-service", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	go func() {
		log.Info("Starting HTTP server", zap.String("port", cfg.ServerPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	go func() {
		grpcAddr := fmt.Sprintf(":%s", cfg.GRPCPort)
		listener, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatal("Failed to listen for gRPC", zap.Error(err))
		}

		log.Info("Starting gRPC server", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal("Failed to start gRPC server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down servers...")

	healthServer.SetServingStatus("category-service", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		log.Warn("Timeout during gRPC server shutdown, forcing stop")
		grpcServer.Stop()
	case <-stopped:
		log.Info("gRPC server stopped gracefully")
	}

	log.Info("Servers exited properly")
}
