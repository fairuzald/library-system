package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/fairuzald/library-system/pkg/logger"
	"go.uber.org/zap"
)

type ServiceProxy struct {
	services map[string]*httputil.ReverseProxy
	log      *logger.Logger
}

type Config struct {
	BookServiceURL     string
	CategoryServiceURL string
	UserServiceURL     string
	Logger             *logger.Logger
}

func NewServiceProxy(config *Config) (*ServiceProxy, error) {
	if config.Logger == nil {
		config.Logger = logger.Default()
	}

	// Initialize services map
	services := make(map[string]*httputil.ReverseProxy)

	// Book service
	if config.BookServiceURL != "" {
		target, err := url.Parse(formatURL(config.BookServiceURL))
		if err != nil {
			return nil, err
		}
		services["book"] = createReverseProxy(target, config.Logger)
	}

	// Category service
	if config.CategoryServiceURL != "" {
		target, err := url.Parse(formatURL(config.CategoryServiceURL))
		if err != nil {
			return nil, err
		}
		services["category"] = createReverseProxy(target, config.Logger)
	}

	// User service
	if config.UserServiceURL != "" {
		target, err := url.Parse(formatURL(config.UserServiceURL))
		if err != nil {
			return nil, err
		}
		services["user"] = createReverseProxy(target, config.Logger)
	}

	return &ServiceProxy{
		services: services,
		log:      config.Logger,
	}, nil
}

// RegisterHandlers registers handlers for each service
func (sp *ServiceProxy) RegisterHandlers(mux *http.ServeMux) {
	// Book service handlers
	if proxy, ok := sp.services["book"]; ok {
		mux.HandleFunc("/api/books/", func(w http.ResponseWriter, r *http.Request) {
			sp.log.Debug("Proxying to book service", zap.String("path", r.URL.Path))
			proxy.ServeHTTP(w, r)
		})
	}

	// Category service handlers
	if proxy, ok := sp.services["category"]; ok {
		mux.HandleFunc("/api/categories/", func(w http.ResponseWriter, r *http.Request) {
			sp.log.Debug("Proxying to category service", zap.String("path", r.URL.Path))
			proxy.ServeHTTP(w, r)
		})
	}

	// User service handlers
	if proxy, ok := sp.services["user"]; ok {
		mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
			sp.log.Debug("Proxying to user service", zap.String("path", r.URL.Path))
			proxy.ServeHTTP(w, r)
		})

		mux.HandleFunc("/api/auth/", func(w http.ResponseWriter, r *http.Request) {
			sp.log.Debug("Proxying auth request to user service", zap.String("path", r.URL.Path))
			proxy.ServeHTTP(w, r)
		})
	}
}

func createReverseProxy(target *url.URL, log *logger.Logger) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the director to modify outgoing requests
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Preserve the original request ID if it exists
		if requestID := req.Header.Get("X-Request-ID"); requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}

		// Add a custom header to identify the gateway
		req.Header.Set("X-Forwarded-By", "api-gateway")
	}

	// Set up custom transport for timeouts
	proxy.Transport = &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	// Handle errors
	proxy.ErrorLog = zap.NewStdLog(log.Logger)

	// Define error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("Proxy error",
			zap.String("url", r.URL.String()),
			zap.String("method", r.Method),
			zap.Error(err),
		)

		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Service temporarily unavailable"))
	}

	return proxy
}

func formatURL(serviceURL string) string {
	if !strings.HasPrefix(serviceURL, "http://") && !strings.HasPrefix(serviceURL, "https://") {
		return "http://" + serviceURL
	}
	return serviceURL
}
