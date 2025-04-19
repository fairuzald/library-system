package routes

import (
	"github.com/fairuzald/library-system/pkg/config"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/middleware"
	"github.com/fairuzald/library-system/services/user-service/internal/handler"
	"github.com/gorilla/mux"
)

func SetupRoutes(
	router *mux.Router,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	jwtAuth *middleware.JWTAuth,
	log *logger.Logger,
	cfg *config.Config,
) {
	apiRouter := router.PathPrefix("/api").Subrouter()

	authRouter := apiRouter.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/login", authHandler.HandleLogin).Methods("POST")
	authRouter.HandleFunc("/register", authHandler.HandleRegister).Methods("POST")
	authRouter.HandleFunc("/refresh", authHandler.HandleRefreshToken).Methods("POST")

	authProtectedRouter := authRouter.NewRoute().Subrouter()
	authProtectedRouter.Use(jwtAuth.HTTPMiddleware)
	authProtectedRouter.HandleFunc("/logout", authHandler.HandleLogout).Methods("POST")

	userRouter := apiRouter.PathPrefix("/users").Subrouter()

	userProtectedRouter := userRouter.NewRoute().Subrouter()
	userProtectedRouter.Use(jwtAuth.HTTPMiddleware)

	userProtectedRouter.HandleFunc("", userHandler.HandleCreateUser).Methods("POST")
	userProtectedRouter.HandleFunc("", userHandler.HandleListUsers).Methods("GET")

	userProtectedRouter.HandleFunc("/{id}", userHandler.HandleGetUser).Methods("GET")
	userProtectedRouter.HandleFunc("/{id}", userHandler.HandleUpdateUser).Methods("PUT", "PATCH")
	userProtectedRouter.HandleFunc("/{id}", userHandler.HandleDeleteUser).Methods("DELETE")
	userProtectedRouter.HandleFunc("/{id}/password", userHandler.HandleChangePassword).Methods("PUT")

	userProtectedRouter.HandleFunc("/email", userHandler.HandleGetUserByEmail).Methods("GET")
	userProtectedRouter.HandleFunc("/username", userHandler.HandleGetUserByUsername).Methods("GET")
}
