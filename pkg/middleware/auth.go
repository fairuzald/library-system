package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/fairuzald/library-system/pkg/utils"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ContextKey string

const (
	UserIDKey     ContextKey = "user_id"
	UserRoleKey   ContextKey = "user_role"
	UserEmailKey  ContextKey = "user_email"
	AuthTokenKey  ContextKey = "auth_token"
	AuthHeaderKey            = "Authorization"
	BearerSchema             = "Bearer"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JWTAuth struct {
	secretKey     []byte
	tokenDuration time.Duration
}

func NewJWTAuth(secretKey string, tokenDuration time.Duration) *JWTAuth {
	return &JWTAuth{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

func (j *JWTAuth) GenerateToken(userID, email, role, username string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Email:    email,
		Role:     role,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTAuth) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected token signing method")
			}
			return j.secretKey, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (j *JWTAuth) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get(AuthHeaderKey)
		if authHeader == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "authorization header is required", nil)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != BearerSchema {
			utils.RespondWithError(w, http.StatusUnauthorized, "invalid authorization header format", nil)
			return
		}

		claims, err := j.ValidateToken(parts[1])
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token", err)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
		ctx = context.WithValue(ctx, AuthTokenKey, parts[1])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (j *JWTAuth) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata is not provided")
		}

		values := md.Get(AuthHeaderKey)
		if len(values) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
		}

		authHeader := values[0]
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != BearerSchema {
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization header format")
		}

		claims, err := j.ValidateToken(parts[1])
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid or expired token: %v", err)
		}

		newCtx := context.WithValue(ctx, UserIDKey, claims.UserID)
		newCtx = context.WithValue(newCtx, UserRoleKey, claims.Role)
		newCtx = context.WithValue(newCtx, UserEmailKey, claims.Email)
		newCtx = context.WithValue(newCtx, AuthTokenKey, parts[1])

		return handler(newCtx, req)
	}
}

func (j *JWTAuth) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Errorf(codes.Unauthenticated, "metadata is not provided")
		}

		values := md.Get(AuthHeaderKey)
		if len(values) == 0 {
			return status.Errorf(codes.Unauthenticated, "authorization token is not provided")
		}

		authHeader := values[0]
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != BearerSchema {
			return status.Errorf(codes.Unauthenticated, "invalid authorization header format")
		}

		claims, err := j.ValidateToken(parts[1])
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "invalid or expired token: %v", err)
		}

		newCtx := context.WithValue(ctx, UserIDKey, claims.UserID)
		newCtx = context.WithValue(newCtx, UserRoleKey, claims.Role)
		newCtx = context.WithValue(newCtx, UserEmailKey, claims.Email)
		newCtx = context.WithValue(newCtx, AuthTokenKey, parts[1])

		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          newCtx,
		}

		return handler(srv, wrappedStream)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
