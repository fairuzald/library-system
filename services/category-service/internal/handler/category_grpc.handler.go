package handler

import (
	"context"
	"strings"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/proto/category"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/category-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/category-service/internal/service"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CategoryGRPCHandler struct {
	category.UnimplementedCategoryServiceServer
	categoryService service.CategoryService
	log             *logger.Logger
}

func NewCategoryGRPCHandler(categoryService service.CategoryService, log *logger.Logger) *CategoryGRPCHandler {
	return &CategoryGRPCHandler{
		categoryService: categoryService,
		log:             log,
	}
}

func (h *CategoryGRPCHandler) GetCategory(ctx context.Context, req *category.GetCategoryRequest) (*category.CategoryResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category ID")
	}

	categoryResponse, err := h.categoryService.GetCategoryByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrCategoryNotFound)
		}
		h.log.Error("Failed to get category", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &category.CategoryResponse{
		Category: convertDaoCategoryToProtoCategory(categoryResponse),
	}, nil
}

func (h *CategoryGRPCHandler) GetCategoryByName(ctx context.Context, req *category.GetCategoryByNameRequest) (*category.CategoryResponse, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	categoryResponse, err := h.categoryService.GetCategoryByName(ctx, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		h.log.Error("Failed to get category by name", zap.Error(err), zap.String("name", name))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &category.CategoryResponse{
		Category: convertDaoCategoryToProtoCategory(categoryResponse),
	}, nil
}

func (h *CategoryGRPCHandler) ListCategories(ctx context.Context, req *category.ListCategoriesRequest) (*category.ListCategoriesResponse, error) {
	filter := &dto.CategoryFilter{
		Page:   int(req.GetPage()),
		Limit:  int(req.GetPageSize()),
		SortBy: req.GetSortBy(),
		Desc:   req.GetSortDesc(),
	}

	if req.ParentId != nil {
		parentID := req.GetParentId()
		filter.ParentID = &parentID
	}

	response, err := h.categoryService.ListCategories(ctx, filter)
	if err != nil {
		h.log.Error("Failed to list categories", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	protoResponse := &category.ListCategoriesResponse{
		Categories:  make([]*category.Category, 0, len(response.Categories)),
		TotalItems:  int64(response.TotalItems),
		TotalPages:  int32(response.TotalPages),
		CurrentPage: int32(response.CurrentPage),
		PageSize:    int32(response.PageSize),
	}

	for _, c := range response.Categories {
		protoResponse.Categories = append(protoResponse.Categories, convertCategoryResponseToProtoCategory(&c))
	}

	return protoResponse, nil
}

func (h *CategoryGRPCHandler) CreateCategory(ctx context.Context, req *category.CreateCategoryRequest) (*category.CategoryResponse, error) {
	createDTO := &dto.CategoryCreate{
		Name:        req.GetName(),
		Description: req.GetDescription(),
	}

	if req.ParentId != nil {
		parentID := req.GetParentId()
		createDTO.ParentID = &parentID
	}

	categoryResponse, err := h.categoryService.CreateCategory(ctx, createDTO)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		if strings.Contains(err.Error(), "invalid parent ID") || strings.Contains(err.Error(), "parent category not found") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		h.log.Error("Failed to create category", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &category.CategoryResponse{
		Category: convertDaoCategoryToProtoCategory(categoryResponse),
	}, nil
}

func (h *CategoryGRPCHandler) UpdateCategory(ctx context.Context, req *category.UpdateCategoryRequest) (*category.CategoryResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category ID")
	}

	updateDTO := &dto.CategoryUpdate{}

	if req.Name != nil {
		name := req.GetName()
		updateDTO.Name = &name
	}

	if req.Description != nil {
		description := req.GetDescription()
		updateDTO.Description = &description
	}

	if req.ParentId != nil {
		parentID := req.GetParentId()
		updateDTO.ParentID = &parentID
	}

	categoryResponse, err := h.categoryService.UpdateCategory(ctx, id, updateDTO)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrCategoryNotFound)
		}
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		if strings.Contains(err.Error(), "invalid parent ID") || strings.Contains(err.Error(), "parent category not found") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if strings.Contains(err.Error(), "cannot be its own parent") || strings.Contains(err.Error(), "create a category cycle") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		h.log.Error("Failed to update category", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &category.CategoryResponse{
		Category: convertDaoCategoryToProtoCategory(categoryResponse),
	}, nil
}

func (h *CategoryGRPCHandler) DeleteCategory(ctx context.Context, req *category.DeleteCategoryRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid category ID")
	}

	if err := h.categoryService.DeleteCategory(ctx, id); err != nil {
		if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrCategoryNotFound)
		}
		if strings.Contains(err.Error(), "child categories") || strings.Contains(err.Error(), "associated books") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		h.log.Error("Failed to delete category", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &emptypb.Empty{}, nil
}

func (h *CategoryGRPCHandler) GetCategoryChildren(ctx context.Context, req *category.GetCategoryChildrenRequest) (*category.ListCategoriesResponse, error) {
	parentID, err := uuid.Parse(req.GetParentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid parent ID")
	}

	response, err := h.categoryService.GetCategoryChildren(ctx, parentID)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrCategoryNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrCategoryNotFound)
		}
		h.log.Error("Failed to get category children", zap.Error(err), zap.String("parent_id", parentID.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	protoResponse := &category.ListCategoriesResponse{
		Categories:  make([]*category.Category, 0, len(response.Categories)),
		TotalItems:  int64(response.TotalItems),
		TotalPages:  int32(response.TotalPages),
		CurrentPage: int32(response.CurrentPage),
		PageSize:    int32(response.PageSize),
	}

	for _, c := range response.Categories {
		protoResponse.Categories = append(protoResponse.Categories, convertCategoryResponseToProtoCategory(&c))
	}

	return protoResponse, nil
}

func (h *CategoryGRPCHandler) Health(ctx context.Context, _ *emptypb.Empty) (*category.HealthResponse, error) {
	return &category.HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}

func convertCategoryResponseToProtoCategory(c *dao.CategoryResponse) *category.Category {
	protoCategory := &category.Category{
		Id:          c.ID.String(),
		Name:        c.Name,
		Description: c.Description,
		CreatedAt:   timestamppb.New(c.CreatedAt),
		UpdatedAt:   timestamppb.New(c.UpdatedAt),
	}

	if c.ParentID != nil {
		parentID := c.ParentID.String()
		protoCategory.ParentId = &parentID
	}

	return protoCategory
}

func convertDaoCategoryToProtoCategory(categoryResponse *dao.CategoryResponse) *category.Category {
	return convertCategoryResponseToProtoCategory(categoryResponse)
}
