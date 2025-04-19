package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fairuzald/library-system/pkg/config"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/swaggest/swgui"
	v5 "github.com/swaggest/swgui/v5"

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

	router.NotFoundHandler = http.HandlerFunc(JSONNotFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(JSONMethodNotAllowed)

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
		recoveryMiddleware.Middleware,
		requestLogger.Middleware,
		rateLimiter.Middleware,
	)

	staticDir := "./static"
	if _, err := os.Stat(filepath.Join(staticDir, "openapi.json")); err != nil {
		staticDir, err = filepath.Abs("./static")
		if err != nil {
			log.Error("Failed to resolve static directory absolute path", zap.Error(err))
		}

		alternatePaths := []string{
			"/app/static",
			"../static",
			"../../static",
			"api-gateway/static",
		}

		found := false
		for _, path := range alternatePaths {
			if _, err := os.Stat(filepath.Join(path, "openapi.json")); err == nil {
				staticDir = path
				found = true
				log.Info("Found openapi.json in alternate path", zap.String("path", staticDir))
				break
			}
		}

		if !found {
			log.Warn("OpenAPI JSON file not found in any expected directories")
		}
	}

	log.Info("Using static directory path", zap.String("path", staticDir))
	files, err := filepath.Glob(filepath.Join(staticDir, "*"))
	if err != nil {
		log.Error("Failed to list files in static directory", zap.Error(err))
	} else {
		log.Info("Files in static directory", zap.Strings("files", files))
	}

	router.HandleFunc("/debug/static", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Static directory: %s\n\n", staticDir)

		files, err := filepath.Glob(filepath.Join(staticDir, "*"))
		if err != nil {
			fmt.Fprintf(w, "Error listing files: %v\n", err)
			return
		}

		fmt.Fprintf(w, "Files in static directory:\n")
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				fmt.Fprintf(w, "  %s (error: %v)\n", file, err)
			} else {
				fmt.Fprintf(w, "  %s (size: %d, mode: %s)\n", file, info.Size(), info.Mode())
			}
		}

		openAPIPath := filepath.Join(staticDir, "openapi.json")
		content, err := os.ReadFile(openAPIPath)
		if err != nil {
			fmt.Fprintf(w, "\nError reading openapi.json: %v\n", err)
		} else {
			fmt.Fprintf(w, "\nopenapi.json exists with size: %d bytes\n", len(content))
		}

		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(w, "\nError getting working directory: %v\n", err)
		} else {
			fmt.Fprintf(w, "\nWorking directory: %s\n", wd)
		}
	}).Methods("GET")

	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(staticDir)),
		),
	).Methods("GET")

	swaggerConfig := swgui.Config{
		Title:       "Library API Gateway",
		SwaggerJSON: "/static/openapi.json",
		BasePath:    "/docs",
	}

	log.Info("Swagger configuration",
		zap.String("title", swaggerConfig.Title),
		zap.String("swaggerJSON", swaggerConfig.SwaggerJSON),
		zap.String("basePath", swaggerConfig.BasePath),
	)

	swaggerHandler := v5.NewHandlerWithConfig(swaggerConfig)
	router.PathPrefix("/docs").Handler(swaggerHandler)

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

	setupServiceProxies(router, cfg, log)

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
func JSONNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":       "Not Found",
		"status_code": http.StatusNotFound,
		"message":     "The requested resource was not found",
		"path":        r.URL.Path,
	})
}

func JSONMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":       "Method Not Allowed",
		"status_code": http.StatusMethodNotAllowed,
		"message":     fmt.Sprintf("Method %s is not allowed for this resource", r.Method),
		"path":        r.URL.Path,
	})
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

func setupServiceProxies(router *mux.Router, cfg *APIGatewayConfig, log *logger.Logger) {
	apiRouter := router.PathPrefix("/api").Subrouter()

	bookRouter := apiRouter.PathPrefix("/books").Subrouter()
	bookProxy := createServiceProxy(cfg.BookServiceHTTPURL, log)
	bookRouter.PathPrefix("").Handler(bookProxy)

	categoryRouter := apiRouter.PathPrefix("/categories").Subrouter()
	categoryProxy := createServiceProxy(cfg.CategoryServiceHTTPURL, log)
	categoryRouter.PathPrefix("").Handler(categoryProxy)

	userRouter := apiRouter.PathPrefix("/users").Subrouter()
	userProxy := createServiceProxy(cfg.UserServiceHTTPURL, log)
	userRouter.PathPrefix("").Handler(userProxy)

	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authRouter.PathPrefix("").Handler(userProxy)

	router.NotFoundHandler = http.HandlerFunc(JSONNotFound)
	router.MethodNotAllowedHandler = http.HandlerFunc(JSONMethodNotAllowed)
}

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

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetURL := fmt.Sprintf("%s%s", target, r.URL.Path)
		if r.URL.RawQuery != "" {
			targetURL = fmt.Sprintf("%s?%s", targetURL, r.URL.RawQuery)
		}

		proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
		if err != nil {
			log.Error("Failed to create proxy request",
				zap.Error(err),
				zap.String("original_url", r.URL.String()),
				zap.String("target_url", targetURL),
			)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		proxyReq.Header.Set("X-Forwarded-Host", r.Host)
		proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
		proxyReq.Header.Set("X-Gateway", "library-system-api-gateway")

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Error("Proxy error",
				zap.Error(err),
				zap.String("url", r.URL.String()),
				zap.String("method", r.Method),
			)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}

		w.WriteHeader(resp.StatusCode)

		if _, err := copyBuffer(w, resp.Body); err != nil {
			log.Error("Failed to copy response body",
				zap.Error(err),
				zap.String("url", r.URL.String()),
			)
		}
	})
}

func copyBuffer(dst http.ResponseWriter, src io.ReadCloser) (int64, error) {
	var buf = make([]byte, 32*1024)
	var written int64
	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, err
		}
	}
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

	rateLimitIP, err := getEnvAsFloat("RATE_LIMIT_IP", 10)
	if err != nil {
		return nil, err
	}

	rateLimitIPBurst, err := getEnvAsInt("RATE_LIMIT_IP_BURST", 20)
	if err != nil {
		return nil, err
	}

	rateLimitGlobal, err := getEnvAsFloat("RATE_LIMIT_GLOBAL", 100)
	if err != nil {
		return nil, err
	}

	rateLimitGBurst, err := getEnvAsInt("RATE_LIMIT_GLOBAL_BURST", 200)
	if err != nil {
		return nil, err
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

func getEnvAsInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %v", key, err)
	}

	return intValue, nil
}

func getEnvAsFloat(key string, defaultValue float64) (float64, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %v", key, err)
	}

	return floatValue, nil
}
