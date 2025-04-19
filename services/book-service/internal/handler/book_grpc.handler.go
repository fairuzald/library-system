package handler

import (
	"context"
	"strings"

	"github.com/fairuzald/library-system/pkg/constants"
	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/proto/book"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/dao"
	"github.com/fairuzald/library-system/services/book-service/internal/entity/dto"
	"github.com/fairuzald/library-system/services/book-service/internal/service"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BookGRPCHandler struct {
	book.UnimplementedBookServiceServer
	bookService service.BookService
	log         *logger.Logger
}

func NewBookGRPCHandler(bookService service.BookService, log *logger.Logger) *BookGRPCHandler {
	return &BookGRPCHandler{
		bookService: bookService,
		log:         log,
	}
}

func (h *BookGRPCHandler) GetBook(ctx context.Context, req *book.GetBookRequest) (*book.BookResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid book ID")
	}

	bookResponse, err := h.bookService.GetBookByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrBookNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrBookNotFound)
		}
		h.log.Error("Failed to get book", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &book.BookResponse{
		Book: convertDaoBookToProtoBook(bookResponse),
	}, nil
}

func (h *BookGRPCHandler) ListBooks(ctx context.Context, req *book.ListBooksRequest) (*book.ListBooksResponse, error) {
	filter := &dto.BookFilter{
		Page:   int(req.GetPage()),
		Limit:  int(req.GetPageSize()),
		SortBy: req.GetSortBy(),
		Desc:   req.GetSortDesc(),
	}

	response, err := h.bookService.ListBooks(ctx, filter)
	if err != nil {
		h.log.Error("Failed to list books", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	protoResponse := &book.ListBooksResponse{
		Books:       make([]*book.Book, 0, len(response.Books)),
		TotalItems:  int64(response.TotalItems),
		TotalPages:  int32(response.TotalPages),
		CurrentPage: int32(response.CurrentPage),
		PageSize:    int32(response.PageSize),
	}

	for _, b := range response.Books {
		protoResponse.Books = append(protoResponse.Books, convertBookResponseToProtoBook(&b))
	}

	return protoResponse, nil
}

func (h *BookGRPCHandler) CreateBook(ctx context.Context, req *book.CreateBookRequest) (*book.BookResponse, error) {
	createDTO := &dto.BookCreate{
		Title:         req.GetTitle(),
		Author:        req.GetAuthor(),
		ISBN:          req.GetIsbn(),
		PublishedYear: int(req.GetPublishedYear()),
		Publisher:     req.GetPublisher(),
		Description:   req.GetDescription(),
		CategoryIDs:   req.GetCategoryIds(),
		Language:      req.GetLanguage(),
		PageCount:     int(req.GetPageCount()),
	}

	if req.GetCoverImage() != "" {
		createDTO.CoverImage = req.GetCoverImage()
	}

	if req.GetQuantity() > 0 {
		createDTO.Quantity = int(req.GetQuantity())
	}

	bookResponse, err := h.bookService.CreateBook(ctx, createDTO)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		if strings.Contains(err.Error(), "invalid category ID") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		h.log.Error("Failed to create book", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &book.BookResponse{
		Book: convertDaoBookToProtoBook(bookResponse),
	}, nil
}

func (h *BookGRPCHandler) UpdateBook(ctx context.Context, req *book.UpdateBookRequest) (*book.BookResponse, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid book ID")
	}

	updateDTO := &dto.BookUpdate{}

	if req.Title != nil {
		title := req.GetTitle()
		updateDTO.Title = &title
	}

	if req.Author != nil {
		author := req.GetAuthor()
		updateDTO.Author = &author
	}

	if req.Isbn != nil {
		isbn := req.GetIsbn()
		updateDTO.ISBN = &isbn
	}

	if req.PublishedYear != nil {
		publishedYear := int(req.GetPublishedYear())
		updateDTO.PublishedYear = &publishedYear
	}

	if req.Publisher != nil {
		publisher := req.GetPublisher()
		updateDTO.Publisher = &publisher
	}

	if req.Description != nil {
		description := req.GetDescription()
		updateDTO.Description = &description
	}

	if len(req.GetCategoryIds()) > 0 {
		updateDTO.CategoryIDs = req.GetCategoryIds()
	}

	if req.Language != nil {
		language := req.GetLanguage()
		updateDTO.Language = &language
	}

	if req.PageCount != nil {
		pageCount := int(req.GetPageCount())
		updateDTO.PageCount = &pageCount
	}

	if req.Status != nil {
		status := req.GetStatus()
		updateDTO.Status = &status
	}

	if req.CoverImage != nil {
		coverImage := req.GetCoverImage()
		updateDTO.CoverImage = &coverImage
	}

	if req.Quantity != nil {
		quantity := int(req.GetQuantity())
		updateDTO.Quantity = &quantity
	}

	if req.AvailableQuantity != nil {
		availableQuantity := int(req.GetAvailableQuantity())
		updateDTO.AvailableQuantity = &availableQuantity
	}

	bookResponse, err := h.bookService.UpdateBook(ctx, id, updateDTO)
	if err != nil {
		if strings.Contains(err.Error(), constants.ErrBookNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrBookNotFound)
		}
		if strings.Contains(err.Error(), "already exists") {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		if strings.Contains(err.Error(), "invalid category ID") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		h.log.Error("Failed to update book", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &book.BookResponse{
		Book: convertDaoBookToProtoBook(bookResponse),
	}, nil
}

func (h *BookGRPCHandler) DeleteBook(ctx context.Context, req *book.DeleteBookRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid book ID")
	}

	if err := h.bookService.DeleteBook(ctx, id); err != nil {
		if strings.Contains(err.Error(), constants.ErrBookNotFound) {
			return nil, status.Error(codes.NotFound, constants.ErrBookNotFound)
		}
		h.log.Error("Failed to delete book", zap.Error(err), zap.String("id", id.String()))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	return &emptypb.Empty{}, nil
}

func (h *BookGRPCHandler) SearchBooks(ctx context.Context, req *book.SearchBooksRequest) (*book.ListBooksResponse, error) {
	search := &dto.BookSearch{
		Query: req.GetQuery(),
		Page:  int(req.GetPage()),
		Limit: int(req.GetPageSize()),
	}

	if req.Field != nil {
		search.Field = req.GetField()
	}

	response, err := h.bookService.SearchBooks(ctx, search)
	if err != nil {
		h.log.Error("Failed to search books", zap.Error(err))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	protoResponse := &book.ListBooksResponse{
		Books:       make([]*book.Book, 0, len(response.Books)),
		TotalItems:  int64(response.TotalItems),
		TotalPages:  int32(response.TotalPages),
		CurrentPage: int32(response.CurrentPage),
		PageSize:    int32(response.PageSize),
	}

	for _, b := range response.Books {
		protoResponse.Books = append(protoResponse.Books, convertBookResponseToProtoBook(&b))
	}

	return protoResponse, nil
}

func (h *BookGRPCHandler) GetBooksByCategory(ctx context.Context, req *book.GetBooksByCategoryRequest) (*book.ListBooksResponse, error) {
	categoryID := req.GetCategoryId()
	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())

	response, err := h.bookService.GetBooksByCategory(ctx, categoryID, page, pageSize)
	if err != nil {
		if strings.Contains(err.Error(), "invalid category ID") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if strings.Contains(err.Error(), "does not exist") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		h.log.Error("Failed to get books by category", zap.Error(err), zap.String("category_id", categoryID))
		return nil, status.Error(codes.Internal, constants.ErrInternalServer)
	}

	protoResponse := &book.ListBooksResponse{
		Books:       make([]*book.Book, 0, len(response.Books)),
		TotalItems:  int64(response.TotalItems),
		TotalPages:  int32(response.TotalPages),
		CurrentPage: int32(response.CurrentPage),
		PageSize:    int32(response.PageSize),
	}

	for _, b := range response.Books {
		protoResponse.Books = append(protoResponse.Books, convertBookResponseToProtoBook(&b))
	}

	return protoResponse, nil
}

func (h *BookGRPCHandler) Health(ctx context.Context, _ *emptypb.Empty) (*book.HealthResponse, error) {
	return &book.HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}

func convertBookResponseToProtoBook(b *dao.BookResponse) *book.Book {
	coverImage := b.CoverImage
	avgRating := float32(b.AverageRating)
	quantity := int32(b.Quantity)
	availableQuantity := int32(b.AvailableQuantity)

	return &book.Book{
		Id:                b.ID.String(),
		Title:             b.Title,
		Author:            b.Author,
		Isbn:              b.ISBN,
		PublishedYear:     int32(b.PublishedYear),
		Publisher:         b.Publisher,
		Description:       b.Description,
		CategoryIds:       b.CategoryIDs,
		Language:          b.Language,
		PageCount:         int32(b.PageCount),
		Status:            b.Status,
		CreatedAt:         timestamppb.New(b.CreatedAt),
		UpdatedAt:         timestamppb.New(b.UpdatedAt),
		CoverImage:        &coverImage,
		AverageRating:     &avgRating,
		Quantity:          &quantity,
		AvailableQuantity: &availableQuantity,
	}
}

func convertDaoBookToProtoBook(bookResponse interface{}) *book.Book {
	switch br := bookResponse.(type) {
	case *dao.BookResponse:
		coverImage := br.CoverImage
		avgRating := float32(br.AverageRating)
		quantity := int32(br.Quantity)
		availableQuantity := int32(br.AvailableQuantity)

		return &book.Book{
			Id:                br.ID.String(),
			Title:             br.Title,
			Author:            br.Author,
			Isbn:              br.ISBN,
			PublishedYear:     int32(br.PublishedYear),
			Publisher:         br.Publisher,
			Description:       br.Description,
			CategoryIds:       br.CategoryIDs,
			Language:          br.Language,
			PageCount:         int32(br.PageCount),
			Status:            br.Status,
			CreatedAt:         timestamppb.New(br.CreatedAt),
			UpdatedAt:         timestamppb.New(br.UpdatedAt),
			CoverImage:        &coverImage,
			AverageRating:     &avgRating,
			Quantity:          &quantity,
			AvailableQuantity: &availableQuantity,
		}
	default:
		return nil
	}
}
