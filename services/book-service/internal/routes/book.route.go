package routes

import (
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/services/book-service/internal/handler"
	"github.com/gorilla/mux"
)

func SetupRoutes(
	router *mux.Router,
	bookHandler *handler.BookHandler,
	jwtAuth *middleware.JWTAuth,
	log *logger.Logger,
) {
	apiRouter := router.PathPrefix("/api").Subrouter()
	booksRouter := apiRouter.PathPrefix("/books").Subrouter()

	// Public routes (no auth required)
	booksRouter.HandleFunc("", bookHandler.HandleListBooks).Methods("GET")
	booksRouter.HandleFunc("/search", bookHandler.HandleSearchBooks).Methods("GET")
	booksRouter.HandleFunc("/isbn", bookHandler.HandleGetBookByISBN).Methods("GET")
	booksRouter.HandleFunc("/category/{categoryId}", bookHandler.HandleGetBooksByCategory).Methods("GET")
	booksRouter.HandleFunc("/{id}", bookHandler.HandleGetBook).Methods("GET")

	// Protected routes (auth required)
	protectedRouter := booksRouter.NewRoute().Subrouter()
	protectedRouter.Use(jwtAuth.HTTPMiddleware)

	protectedRouter.HandleFunc("", bookHandler.HandleCreateBook).Methods("POST")
	protectedRouter.HandleFunc("/{id}", bookHandler.HandleUpdateBook).Methods("PUT", "PATCH")
	protectedRouter.HandleFunc("/{id}", bookHandler.HandleDeleteBook).Methods("DELETE")
}
