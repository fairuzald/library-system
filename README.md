# Library Management System

A scalable microservice architecture for a library management application, featuring book management, category management, user authentication, and search functionality.

## Overview

This project implements a complete library management system with the following microservices:

- **API Gateway**: Central entry point for all client requests
- **Book Service**: Manages book data and operations
- **Category Service**: Handles book categories
- **User Service**: Handles user management and authentication

Each service has its own PostgreSQL database and communicates with other services via gRPC.

## System Design

### Architecture

The system follows a microservice architecture pattern with the following components:

1. **API Gateway**:

   - Acts as a single entry point for all client requests
   - Routes requests to appropriate microservices
   - Handles cross-cutting concerns like authentication, rate limiting, and logging
   - Provides Swagger documentation through `/docs` endpoint

2. **Book Service**:

   - Manages all book-related operations
   - Communicates with Category Service to validate categories
   - Implements search functionality with filters
   - Supports pagination and sorting

3. **Category Service**:

   - Manages book categories with hierarchical relationships
   - Prevents category cycles in hierarchies
   - Ensures referential integrity with books

4. **User Service**:
   - Handles user registration and authentication
   - Manages user profiles and permissions
   - Implements JWT-based authentication
   - Handles token refresh and revocation

### Database Schema (ERD)

```
┌──────────────────┐      ┌────────────────────┐      ┌────────────────────┐
│    Users (User)  │      │    Books (Book)    │      │ Categories (Cat.)  │
├──────────────────┤      ├────────────────────┤      ├────────────────────┤
│ id [PK]          │      │ id [PK]            │      │ id [PK]            │
│ email            │      │ title              │      │ name               │
│ username         │      │ author             │      │ description        │
│ password         │      │ isbn               │      │ parent_id [FK]     │
│ first_name       │      │ published_year     │      │ created_at         │
│ last_name        │      │ publisher          │      │ updated_at         │
│ role             │      │ description        │      │ deleted_at         │
│ status           │      │ language           │      └────────────────────┘
│ phone            │      │ page_count         │              ▲
│ address          │      │ status             │              │
│ last_login       │      │ cover_image        │              │
│ refresh_token    │      │ average_rating     │      ┌───────┴────────────┐
│ refresh_token_exp│      │ quantity           │      │  books_categories  │
│ created_at       │      │ available_quantity │      ├────────────────────┤
│ updated_at       │      │ created_at         │      │ id [PK]            │
│ deleted_at       │      │ updated_at         │      │ book_id [FK]       │
└──────────────────┘      │ deleted_at         │──────┤ category_id [FK]   │
                          └────────────────────┘      │ created_at         │
                                                      │ updated_at         │
                                                      └────────────────────┘
```

### Technology Implementation Details

#### API Gateway

- Built using Gorilla Mux for routing
- Implements middleware for:
  - Request logging with unique request IDs
  - Rate limiting (both per-IP and global)
  - Recovery from panics
  - CORS handling
- Proxies requests to appropriate microservices
- Health check endpoint that verifies connectivity to all services
- Swagger UI for API documentation

#### Authentication & Authorization

- JWT-based authentication with access and refresh tokens
- Access tokens have short lifespan (15 minutes by default)
- Refresh tokens have longer lifespan (7 days by default)
- Token blacklisting using Redis
- Role-based access control (admin, librarian, member, guest)
- Password hashing using bcrypt

#### Caching

- Redis cache implementation for:
  - Frequently accessed books and categories
  - User data
  - Token blacklisting
- Cache invalidation on updates
- TTL-based cache expiry

#### Database

- PostgreSQL with separate databases for each service
- Database migrations managed with dbmate
- Connection pooling for efficient resource usage
- Soft delete implementation through deleted_at timestamp
- Indexes on frequently queried columns

#### Inter-Service Communication

- gRPC for efficient, type-safe communication between services
- Protocol Buffers for message serialization
- Health checks between services
- Graceful server shutdown
- Keepalive configuration to maintain connections

#### Search and Filtering

- Book search by title, author, ISBN, description
- Category filtering by parent/child relationships
- User filtering by role and status
- Pagination and sorting across all listing endpoints

#### Error Handling

- Consistent error response format across all services
- Appropriate HTTP status codes
- Detailed error messages for developers
- Production-safe error responses that don't expose internals

## Project Structure Details

```
├── api-gateway/           # API Gateway service
│   ├── cmd/               # Entry point
│   ├── internal/          # Internal packages
│   │   └── proxy/         # Proxy implementation
│   ├── static/            # Static files (OpenAPI docs)
│   ├── Dockerfile         # Production Dockerfile
│   └── Dockerfile.dev     # Development Dockerfile
│
├── pkg/                   # Shared packages
│   ├── cache/             # Redis cache implementation
│   ├── config/            # Configuration utilities
│   ├── constants/         # Shared constants
│   ├── database/          # Database utilities
│   ├── logger/            # Logging utilities
│   ├── middleware/        # Shared middlewares
│   │   ├── auth.go        # JWT authentication
│   │   ├── logging.go     # Request logging
│   │   ├── recovery.go    # Panic recovery
│   │   └── ratelimit.go   # Rate limiting
│   ├── models/            # Common models
│   └── utils/             # Helper functions
│       ├── hash.go        # Password hashing
│       ├── response.go    # HTTP response helpers
│       └── validation.go  # Input validation
│
├── proto/                 # Protocol buffer definitions
│   ├── book/              # Book service proto
│   ├── category/          # Category service proto
│   └── user/              # User service proto
│
├── services/              # Services
│   ├── book-service/      # Book service
│   │   ├── cmd/           # Entry point
│   │   ├── internal/      # Internal packages
│   │   │   ├── entity/    # Domain entities
│   │   │   ├── handler/   # HTTP and gRPC handlers
│   │   │   ├── module/    # Service modules
│   │   │   ├── repository/# Data access layer
│   │   │   ├── routes/    # HTTP routes
│   │   │   └── service/   # Business logic
│   │   ├── Dockerfile     # Production Dockerfile
│   │   └── Dockerfile.dev # Development Dockerfile
│   │
│   ├── category-service/  # Category service (similar structure)
│   └── user-service/      # User service (similar structure)
│
├── migrations/            # Database migrations
│   ├── book/              # Book service migrations
│   ├── category/          # Category service migrations
│   └── user/              # User service migrations
│
├── deployment/            # Deployment configurations
│   └── docker/            # Docker compose files
│       ├── docker-compose.dev.yml  # Development environment
│       └── docker-compose.yml      # Production environment
│
├── .env.dev              # Development environment variables
├── .env.prod             # Production environment variables
├── Makefile              # Build and deployment scripts
└── README.md             # Project documentation
```

## API Endpoints

The system exposes RESTful APIs through the API Gateway:

### Authentication

- `POST /api/auth/register`: Register a new user
- `POST /api/auth/login`: Login and get JWT token
- `POST /api/auth/refresh`: Refresh access token
- `POST /api/auth/logout`: Logout (requires authentication)

### Books

- `GET /api/books`: List all books with pagination
- `GET /api/books/{id}`: Get book by ID
- `POST /api/books`: Create a new book (requires auth)
- `PUT /api/books/{id}`: Update a book (requires auth)
- `DELETE /api/books/{id}`: Delete a book (requires auth)
- `GET /api/books/search`: Search books by query

### Categories

- `GET /api/categories`: List all categories
- `GET /api/categories/{id}`: Get category by ID
- `POST /api/categories`: Create a new category (requires auth)
- `PUT /api/categories/{id}`: Update a category (requires auth)
- `DELETE /api/categories/{id}`: Delete a category (requires auth)

### Users

- `GET /api/users/{id}`: Get user information (requires auth)
- `PUT /api/users/{id}`: Update user information (requires auth)
- `DELETE /api/users/{id}`: Delete user (admin only)
- `PUT /api/users/{id}/password`: Change password (requires auth)

## Getting Started

### Step-by-Step Setup

1. **Clone the repository**:

   ```bash
   git clone https://github.com/your-username/library-management-system.git
   cd library-management-system
   ```

2. **Set up environment variables**:

   ```bash
   # Copy environment templates
   cp .env.example.dev .env.dev
   cp .env.example.prod .env.prod

   # Edit the files with your desired configuration
   nano .env.dev
   ```

   Essential environment variables to configure:

   - `JWT_SECRET`: A secure random string for JWT token signing
   - Database credentials for each service
   - Redis connection details

3. **Install dbmate** (if not using Makefile):

   ```bash
   # Linux
   curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64
   chmod +x /usr/local/bin/dbmate

   # macOS
   brew install dbmate
   ```

4. **Start development environment**:

   Using Makefile:

   ```bash
   make dev
   ```

   Without Makefile:

   ```bash
   docker-compose -f deployment/docker/docker-compose.dev.yml --env-file .env.dev up -d
   ```

5. **Run database migrations**:

   Using Makefile:

   ```bash
   make migrate-dev
   ```

   Without Makefile:

   ```bash
   # Book service
   DATABASE_URL=postgres://${BOOK_DB_USER}:${BOOK_DB_PASSWORD}@localhost:${BOOK_DB_EXPOSE_PORT}/${BOOK_DB_NAME}?sslmode=${DB_SSLMODE} dbmate -d migrations/book up

   # Category service
   DATABASE_URL=postgres://${CATEGORY_DB_USER}:${CATEGORY_DB_PASSWORD}@localhost:${CATEGORY_DB_EXPOSE_PORT}/${CATEGORY_DB_NAME}?sslmode=${DB_SSLMODE} dbmate -d migrations/category up

   # User service
   DATABASE_URL=postgres://${USER_DB_USER}:${USER_DB_PASSWORD}@localhost:${USER_DB_EXPOSE_PORT}/${USER_DB_NAME}?sslmode=${DB_SSLMODE} dbmate -d migrations/user up
   ```

6. **Verify the services**:

   Check service health:

   ```bash
   curl http://localhost:8000/health
   ```

   The API Gateway will be available at: `http://localhost:8000`

   Swagger Documentation: `http://localhost:8000/docs`

7. **Stop the services**:

   Using Makefile:

   ```bash
   make down-dev
   ```

   Without Makefile:

   ```bash
   docker-compose -f deployment/docker/docker-compose.dev.yml --env-file .env.dev down
   ```

### Production Deployment

1. **Configure production environment**:

   ```bash
   # Edit production environment variables
   nano .env.prod
   ```

2. **Build and start production containers**:

   Using Makefile:

   ```bash
   # Build images
   make build

   # Start production environment
   make prod
   ```

   Without Makefile:

   ```bash
   # Build images
   docker-compose -f deployment/docker/docker-compose.yml --env-file .env.prod build

   # Start production environment
   docker-compose -f deployment/docker/docker-compose.yml --env-file .env.prod up -d
   ```

3. **Run production migrations**:

   Using Makefile:

   ```bash
   make migrate-prod
   ```

   Without Makefile:

   ```bash
   # Similar to development migrations but using production database credentials
   ```

### Useful Makefile Commands

- `make logs-api`: View API Gateway logs
- `make logs-book`: View Book service logs
- `make logs-category`: View Category service logs
- `make logs-user`: View User service logs
- `make logs-all`: View all service logs
- `make clean`: Remove all containers, volumes, and networks
- `make build`: Build Docker images
- `make push`: Push Docker images to Docker Hub
- `make sql-book`: Connect to book database using psql
- `make sql-category`: Connect to category database using psql
- `make sql-user`: Connect to user database using psql

## Docker Images

All services are containerized and available on Docker Hub:

- [fairuzald/library-api-gateway](https://hub.docker.com/repository/docker/fairuzald/library-api-gateway)
- [fairuzald/library-book-service](https://hub.docker.com/repository/docker/fairuzald/library-book-service)
- [fairuzald/library-category-service](https://hub.docker.com/repository/docker/fairuzald/library-category-service)
- [fairuzald/library-user-service](https://hub.docker.com/repository/docker/fairuzald/library-user-service)

## Implementation Features

### Security

- Request rate limiting to prevent abuse
- Password hashing with bcrypt
- JWT token expiration and renewal
- Token blacklisting after logout
- HTTPS-ready configuration (just add certificates)
- Role-based access control
- Input validation and sanitization

### Reliability

- Graceful server shutdown
- Error recovery middleware
- Comprehensive logging
- Health checks for all services
- Database connection pooling
- Request timeouts

### Performance

- Redis caching for frequently accessed data
- Efficient gRPC communication between services
- Connection pooling for databases
- Database indexing on frequently queried columns
- Pagination for all list endpoints
- Optimized PostgreSQL queries

### Scalability

- Containerized services for easy scaling
- Each service can be scaled independently
- Stateless design for horizontal scaling
- Independent databases per service
- Redis for distributed caching

## Notes from the Developer

Due to time constraints, i already contact to email about my problem, so (only 1 day to complete the project), there are some limitations:

1. **Unit Tests**: I wasn't able to implement comprehensive unit tests as required in the specifications.

2. **Cloud Deployment**: I encountered an issue with my payment method when attempting to deploy to cloud services.

3. **Race Conditions**: While the system includes basic protections against race conditions through database transactions, more advanced techniques would require further development.

Despite these limitations, the core functionality is fully implemented, including:

- Microservice architecture with proper separation of concerns
- Containerization with Docker
- Inter-service communication with gRPC
- JWT authentication
- Redis caching
- Comprehensive API endpoints

Thank you for reviewing my submission!
