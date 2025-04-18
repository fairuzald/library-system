package middleware

import (
	"context"
	"time"

	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func UnaryLoggingInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		requestID := uuid.New().String()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		md.Set("x-request-id", requestID)
		newCtx := metadata.NewIncomingContext(ctx, md)

		p, ok := peer.FromContext(ctx)
		clientIP := "unknown"
		if ok {
			clientIP = p.Addr.String()
		}

		header := metadata.Pairs("x-request-id", requestID)
		grpc.SetHeader(ctx, header)

		resp, err := handler(newCtx, req)

		duration := time.Since(start)
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("grpc_method", info.FullMethod),
			zap.String("client_ip", clientIP),
			zap.Duration("duration", duration),
			zap.Int64("duration_ms", duration.Milliseconds()),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Error("gRPC request failed", fields...)
		} else {
			log.Info("gRPC request", fields...)
		}

		return resp, err
	}
}

func StreamLoggingInterceptor(log *logger.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		requestID := uuid.New().String()

		ctx := ss.Context()
		p, ok := peer.FromContext(ctx)
		clientIP := "unknown"
		if ok {
			clientIP = p.Addr.String()
		}

		err := handler(srv, &wrappedStream{
			ServerStream: ss,
			requestID:    requestID,
		})

		duration := time.Since(start)
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("grpc_method", info.FullMethod),
			zap.String("client_ip", clientIP),
			zap.Bool("is_stream", true),
			zap.Duration("duration", duration),
			zap.Int64("duration_ms", duration.Milliseconds()),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			log.Error("gRPC stream failed", fields...)
		} else {
			log.Info("gRPC stream", fields...)
		}

		return err
	}
}

type wrappedStream struct {
	grpc.ServerStream
	requestID string
}

func (w *wrappedStream) Context() context.Context {
	ctx := w.ServerStream.Context()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	md.Set("x-request-id", w.requestID)
	return metadata.NewIncomingContext(ctx, md)
}
