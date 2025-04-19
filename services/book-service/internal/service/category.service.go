package service

import (
	"context"
	"time"

	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/proto/category"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcCategoryClient struct {
	conn   *grpc.ClientConn
	client category.CategoryServiceClient
	log    *logger.Logger
}

// NewCategoryClient creates a new client for the Category service
func NewCategoryClient(serviceURL string, log *logger.Logger) (CategoryClient, error) {
	if serviceURL == "" {
		log.Warn("Category service URL is empty, creating mock client")
		return &mockCategoryClient{log: log}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Info("Connecting to category service", zap.String("url", serviceURL))
	conn, err := grpc.DialContext(
		ctx,
		serviceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Error("Failed to connect to category service", zap.Error(err), zap.String("url", serviceURL))
		return &mockCategoryClient{log: log}, nil
	}

	client := category.NewCategoryServiceClient(conn)

	return &grpcCategoryClient{
		conn:   conn,
		client: client,
		log:    log,
	}, nil
}

func (c *grpcCategoryClient) CategoryExists(ctx context.Context, categoryID string) (bool, error) {
	req := &category.CheckCategoryExistsRequest{
		Id: categoryID,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := c.client.CheckCategoryExists(ctx, req)
	if err != nil {
		c.log.Error("Failed to check if category exists",
			zap.Error(err),
			zap.String("category_id", categoryID))
		return false, err
	}

	return resp.Exists, nil
}

func (c *grpcCategoryClient) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	req := &category.GetCategoryRequest{
		Id: categoryID,
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := c.client.GetCategory(ctx, req)
	if err != nil {
		c.log.Error("Failed to get category name",
			zap.Error(err),
			zap.String("category_id", categoryID))
		return "", err
	}

	return resp.Category.Name, nil
}

func (c *grpcCategoryClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Mock implementation for when category service is unavailable
type mockCategoryClient struct {
	log *logger.Logger
}

func (m *mockCategoryClient) CategoryExists(ctx context.Context, categoryID string) (bool, error) {
	m.log.Warn("Using mock category client, assuming category exists",
		zap.String("category_id", categoryID))
	return true, nil
}

func (m *mockCategoryClient) GetCategoryName(ctx context.Context, categoryID string) (string, error) {
	m.log.Warn("Using mock category client, returning unknown category",
		zap.String("category_id", categoryID))
	return "Unknown Category", nil
}

func (m *mockCategoryClient) Close() error {
	return nil
}
