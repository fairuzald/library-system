package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RecoveryMiddleware struct {
	log *logger.Logger
}

func NewRecoveryMiddleware(log *logger.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		log: log,
	}
}

func (r *RecoveryMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				r.log.Error("HTTP request panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(stack)),
					zap.String("path", req.URL.Path),
					zap.String("method", req.Method),
				)

				utils.RespondWithError(w, http.StatusInternalServerError, "Internal server error", fmt.Errorf("panic: %v", err))
			}
		}()

		next.ServeHTTP(w, req)
	})
}

func UnaryRecoveryInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				log.Error("gRPC request panic recovered",
					zap.Any("error", r),
					zap.String("stack", string(stack)),
					zap.String("method", info.FullMethod),
				)

				err = status.Errorf(codes.Internal, "internal error")
			}
		}()

		return handler(ctx, req)
	}
}

func StreamRecoveryInterceptor(log *logger.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				log.Error("gRPC stream panic recovered",
					zap.Any("error", r),
					zap.String("stack", string(stack)),
					zap.String("method", info.FullMethod),
				)

				err = status.Errorf(codes.Internal, "internal error")
			}
		}()

		return handler(srv, ss)
	}
}
