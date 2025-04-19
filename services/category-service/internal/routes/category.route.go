package routes

import (
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/services/category-service/internal/handler"
	"github.com/gorilla/mux"
)

func SetupRoutes(
	router *mux.Router,
	categoryHandler *handler.CategoryHandler,
	jwtAuth *middleware.JWTAuth,
	log *logger.Logger,
) {
	apiRouter := router.PathPrefix("/api").Subrouter()
	categoriesRouter := apiRouter.PathPrefix("/categories").Subrouter()

	// Public routes (no auth required)
	categoriesRouter.HandleFunc("", categoryHandler.HandleListCategories).Methods("GET")
	categoriesRouter.HandleFunc("/name", categoryHandler.HandleGetCategoryByName).Methods("GET")
	categoriesRouter.HandleFunc("/{id}", categoryHandler.HandleGetCategory).Methods("GET")
	categoriesRouter.HandleFunc("/{id}/children", categoryHandler.HandleGetCategoryChildren).Methods("GET")

	// Protected routes (auth required)
	protectedRouter := categoriesRouter.NewRoute().Subrouter()
	protectedRouter.Use(jwtAuth.HTTPMiddleware)

	protectedRouter.HandleFunc("", categoryHandler.HandleCreateCategory).Methods("POST")
	protectedRouter.HandleFunc("/{id}", categoryHandler.HandleUpdateCategory).Methods("PUT", "PATCH")
	protectedRouter.HandleFunc("/{id}", categoryHandler.HandleDeleteCategory).Methods("DELETE")
}
