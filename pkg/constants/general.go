package constants

const (
	DefaultPageSize    = 10
	MaxPageSize        = 100
	DefaultSearchLimit = 20

	PasswordMinLength = 8
	UsernameMinLength = 3
	UsernameMaxLength = 30

	TokenTypBearer      = "Bearer"
	HeaderAuthorization = "Authorization"

	// Service names
	ServiceBook     = "book-service"
	ServiceCategory = "category-service"
	ServiceUser     = "user-service"
	ServiceGateway  = "api-gateway"
)
