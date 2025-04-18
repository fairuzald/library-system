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
	"github.com/fairuzald/library-system/services/user-service/internal/handlers"
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
	// Load configuration
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		panic(fmt.Sprintf("Error loading config: %v", err))
	}

	// Initialize logger
	logConfig := config.LoadLoggingConfig()
	log := logger.New(logger.Config{
		Level:      logConfig.Level,
		Production: logConfig.Production,
		JsonFormat: logConfig.JsonFormat,
	})
	defer log.Sync()

	log.Info("Starting user service",
		zap.String("app_name", cfg.AppName),
		zap.String("env", cfg.AppEnv),
		zap.String("version", "1.0.0"),
	)

	// Connect to the database
	db, err := sql.Open("postgres", cfg.GetDSN())
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Set connection pool parameters
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Check database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database", zap.Error(err))
	}
	log.Info("Successfully connected to database", zap.String("database", cfg.DBName))

	// Initialize Redis client
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

	// Initialize JWT auth
	jwtAuth := middleware.NewJWTAuth(cfg.JWTSecret, time.Duration(cfg.JWTExpirationHours)*time.Hour)

	healthHandler := handlers.NewHealthHandler(db, log)

	// Set up HTTP server
	router := mux.NewRouter()

	// Add request logger and recovery middleware
	requestLogger := middleware.NewRequestLogger(log)
	recoveryMiddleware := middleware.NewRecoveryMiddleware(log)
	router.Use(recoveryMiddleware.Middleware, requestLogger.Middleware)

	// Health endpoint
	router.HandleFunc("/health", healthHandler.HandleHealth).Methods("GET")

	// API router
	apiRouter := router.PathPrefix("/api").Subrouter()

	// Auth endpoints (no auth required)
	apiRouter.PathPrefix("/auth").Subrouter()

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Set up gRPC server with interceptors for logging and recovery
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.UnaryRecoveryInterceptor(log),
			middleware.UnaryLoggingInterceptor(log),
			jwtAuth.UnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			middleware.StreamRecoveryInterceptor(log),
			middleware.StreamLoggingInterceptor(log),
			jwtAuth.StreamInterceptor(),
		),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Minute,
			Timeout:               20 * time.Second,
		}),
	)

	// Register user service
	// userProtoService := grpc_handlers.NewUserService(userService, log)
	// user.RegisterUserServiceServer(grpcServer, userProtoService)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for grpcurl
	reflection.Register(grpcServer)

	// Start HTTP server in a goroutine
	go func() {
		log.Info("Starting HTTP server", zap.String("port", cfg.ServerPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Start gRPC server in a goroutine
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

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down servers...")

	// Change health check status to NOT_SERVING
	healthServer.SetServingStatus("user-service", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

	// Gracefully stop gRPC server
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for gRPC server to stop or timeout
	select {
	case <-ctx.Done():
		log.Warn("Timeout during gRPC server shutdown, forcing stop")
		grpcServer.Stop()
	case <-stopped:
		log.Info("gRPC server stopped gracefully")
	}

	log.Info("Servers exited properly")
}
