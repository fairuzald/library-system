package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fairuzald/library-system/pkg/config"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

type APIGatewayConfig struct {
	AppName                string
	AppEnv                 string
	ServerPort             string
	LogLevel               string
	JWTSecret              string
	BookServiceHTTPURL     string
	CategoryServiceHTTPURL string
	UserServiceHTTPURL     string
	BookServiceGRPCURL     string
	CategoryServiceGRPCURL string
	UserServiceGRPCURL     string
	RateLimitIP            float64
	RateLimitIPBurst       int
	RateLimitGlobal        float64
	RateLimitGBurst        int
}

func main() {
	cfg, err := loadAPIGatewayConfig()
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

	log.Info("Starting API gateway",
		zap.String("app_name", cfg.AppName),
		zap.String("env", cfg.AppEnv),
		zap.String("version", "1.0.0"),
	)

	router := mux.NewRouter()

	requestLogger := middleware.NewRequestLogger(log)
	recoveryMiddleware := middleware.NewRecoveryMiddleware(log)

	rateLimiter := middleware.NewRateLimiter(
		log,
		cfg.RateLimitIP,
		cfg.RateLimitIPBurst,
		cfg.RateLimitGlobal,
		cfg.RateLimitGBurst,
	)

	router.Use(
		recoveryMiddleware.Middleware, // First recover from panics
		requestLogger.Middleware,      // Then log the request
		rateLimiter.Middleware,        // Then apply rate limiting
	)

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := HealthResponse{
			Status:    "ok",
			Version:   "1.0.0",
			Timestamp: time.Now(),
			Services: map[string]string{
				"book-service":     "unknown",
				"category-service": "unknown",
				"user-service":     "unknown",
			},
		}

		bookStatus := checkServiceHealth(cfg.BookServiceHTTPURL, log)
		response.Services["book-service"] = bookStatus

		categoryStatus := checkServiceHealth(cfg.CategoryServiceHTTPURL, log)
		response.Services["category-service"] = categoryStatus

		userStatus := checkServiceHealth(cfg.UserServiceHTTPURL, log)
		response.Services["user-service"] = userStatus

		if bookStatus != "ok" || categoryStatus != "ok" || userStatus != "ok" {
			response.Status = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	bookServiceProxy := createServiceProxy(cfg.BookServiceHTTPURL, log)
	categoryServiceProxy := createServiceProxy(cfg.CategoryServiceHTTPURL, log)
	userServiceProxy := createServiceProxy(cfg.UserServiceHTTPURL, log)

	apiRouter := router.PathPrefix("/api").Subrouter()

	bookRouter := apiRouter.PathPrefix("/books").Subrouter()
	bookRouter.PathPrefix("").Handler(bookServiceProxy)

	categoryRouter := apiRouter.PathPrefix("/categories").Subrouter()
	categoryRouter.PathPrefix("").Handler(categoryServiceProxy)

	userRouter := apiRouter.PathPrefix("/users").Subrouter()
	userRouter.PathPrefix("").Handler(userServiceProxy)

	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authRouter.PathPrefix("").Handler(userServiceProxy)

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      corsMiddleware.Handler(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Starting HTTP server", zap.String("port", cfg.ServerPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", zap.Error(err))
	}

	log.Info("Server exited properly")
}

// createServiceProxy creates a new reverse proxy for the specified service
func createServiceProxy(serviceURL string, log *logger.Logger) http.Handler {
	if serviceURL == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Service not configured", http.StatusServiceUnavailable)
		})
	}

	target := serviceURL
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "http://" + target
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			targetURL := fmt.Sprintf("%s%s", target, req.URL.Path)
			if req.URL.RawQuery != "" {
				targetURL = fmt.Sprintf("%s?%s", targetURL, req.URL.RawQuery)
			}

			parsedURL, err := http.NewRequest(req.Method, targetURL, req.Body)
			if err != nil {
				log.Error("Failed to create proxy request",
					zap.Error(err),
					zap.String("original_url", req.URL.String()),
					zap.String("target_url", targetURL),
				)
				return
			}

			for key, values := range req.Header {
				for _, value := range values {
					parsedURL.Header.Add(key, value)
				}
			}

			req.URL = parsedURL.URL
			req.Host = parsedURL.Host

			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			req.Header.Set("X-Forwarded-Proto", "http")
			req.Header.Set("X-Gateway", "library-system-api-gateway")

			log.Debug("Proxying request",
				zap.String("method", req.Method),
				zap.String("original_url", req.URL.String()),
				zap.String("target_url", targetURL),
			)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("Proxy error",
				zap.Error(err),
				zap.String("url", r.URL.String()),
				zap.String("method", r.Method),
			)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		},
	}

	return proxy
}

func checkServiceHealth(serviceURL string, log *logger.Logger) string {
	if serviceURL == "" {
		return "unknown"
	}

	url := serviceURL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	if !strings.HasSuffix(url, "/health") {
		url = strings.TrimSuffix(url, "/") + "/health"
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Error("Failed to check service health", zap.String("url", url), zap.Error(err))
		return "error"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn("Service health check returned non-OK status",
			zap.String("url", url),
			zap.Int("status", resp.StatusCode))
		return "error"
	}

	return "ok"
}

func loadAPIGatewayConfig() (*APIGatewayConfig, error) {
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		return nil, fmt.Errorf("required environment variable not set: APP_NAME")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("required environment variable not set: JWT_SECRET")
	}

	bookServiceHTTPURL := os.Getenv("BOOK_SERVICE_HTTP_URL")
	categoryServiceHTTPURL := os.Getenv("CATEGORY_SERVICE_HTTP_URL")
	userServiceHTTPURL := os.Getenv("USER_SERVICE_HTTP_URL")

	bookServiceGRPCURL := os.Getenv("BOOK_SERVICE_GRPC_URL")
	categoryServiceGRPCURL := os.Getenv("CATEGORY_SERVICE_GRPC_URL")
	userServiceGRPCURL := os.Getenv("USER_SERVICE_GRPC_URL")

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8000"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	rateLimitIP := getEnvAsFloat("RATE_LIMIT_IP", 10)              // 10 req/s per IP
	rateLimitIPBurst := getEnvAsInt("RATE_LIMIT_IP_BURST", 20)     // 20 burst per IP
	rateLimitGlobal := getEnvAsFloat("RATE_LIMIT_GLOBAL", 100)     // 100 req/s global
	rateLimitGBurst := getEnvAsInt("RATE_LIMIT_GLOBAL_BURST", 200) // 200 burst global

	if bookServiceHTTPURL == "" {
		bookServiceHTTPURL = os.Getenv("BOOK_SERVICE_URL")
		if bookServiceHTTPURL != "" && !strings.Contains(bookServiceHTTPURL, ":8080") {
			parts := strings.Split(bookServiceHTTPURL, ":")
			if len(parts) > 0 {
				bookServiceHTTPURL = parts[0] + ":8080"
			}
		}
	}

	if categoryServiceHTTPURL == "" {
		categoryServiceHTTPURL = os.Getenv("CATEGORY_SERVICE_URL")
		if categoryServiceHTTPURL != "" && !strings.Contains(categoryServiceHTTPURL, ":8081") {
			parts := strings.Split(categoryServiceHTTPURL, ":")
			if len(parts) > 0 {
				categoryServiceHTTPURL = parts[0] + ":8081"
			}
		}
	}

	if userServiceHTTPURL == "" {
		userServiceHTTPURL = os.Getenv("USER_SERVICE_URL")
		if userServiceHTTPURL != "" && !strings.Contains(userServiceHTTPURL, ":8082") {
			parts := strings.Split(userServiceHTTPURL, ":")
			if len(parts) > 0 {
				userServiceHTTPURL = parts[0] + ":8082"
			}
		}
	}

	return &APIGatewayConfig{
		AppName:                appName,
		AppEnv:                 appEnv,
		ServerPort:             serverPort,
		LogLevel:               logLevel,
		JWTSecret:              jwtSecret,
		BookServiceHTTPURL:     bookServiceHTTPURL,
		CategoryServiceHTTPURL: categoryServiceHTTPURL,
		UserServiceHTTPURL:     userServiceHTTPURL,
		BookServiceGRPCURL:     bookServiceGRPCURL,
		CategoryServiceGRPCURL: categoryServiceGRPCURL,
		UserServiceGRPCURL:     userServiceGRPCURL,
		RateLimitIP:            rateLimitIP,
		RateLimitIPBurst:       rateLimitIPBurst,
		RateLimitGlobal:        rateLimitGlobal,
		RateLimitGBurst:        rateLimitGBurst,
	}, nil
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}

	return floatValue
}
