# Library Management System

A scalable microservice-based application for managing a library's books, categories, and users.

## Overview

This Library Management System is designed as a modern, cloud-native application using a microservices architecture. It provides RESTful APIs for managing books, book categories, user accounts, and authentication with JWT token-based security.

## Features

- **Microservice Architecture**: Separate services for books, categories, and users
- **REST API**: Full-featured API for all operations
- **gRPC Communication**: Inter-service communication via gRPC
- **Authentication & Authorization**: JWT-based auth with roles (admin, librarian, member)
- **Database**: PostgreSQL with separate databases per service
- **Caching**: Redis for performance optimization
- **Containerization**: Docker and Docker Compose support
- **Documentation**: Complete API documentation

## System Architecture

The system consists of the following components:

### API Gateway

- Routes requests to appropriate microservices
- Handles CORS and rate limiting
- Provides a unified API endpoint for clients

### Microservices

1. **User Service** (Authentication & User Management)

   - User registration, login, and profile management
   - JWT token generation and validation
   - Role-based access control

2. **Book Service** (Book Management)

   - CRUD operations for books
   - Search and filtering capabilities
   - Book availability tracking

3. **Category Service** (Category Management)
   - CRUD operations for book categories
   - Hierarchical category structure
   - Category-book relationships

### Database Layer

- Each service has its own PostgreSQL database
- Data isolation between services
- Optimized schemas with proper indexes

### Communication Patterns

- **Client → Services**: REST API via API Gateway
- **Service → Service**: gRPC for efficient communication
- **Caching**: Redis for frequently accessed data

## Technology Stack

- **Backend**: Go (Golang)
- **API**: RESTful API with JSON
- **Database**: PostgreSQL
- **Caching**: Redis
- **Authentication**: JWT (JSON Web Tokens)
- **Communication**: gRPC for inter-service communication
- **Documentation**: OpenAPI (Swagger)
- **Containerization**: Docker & Docker Compose
- **CI/CD**: GitHub Actions

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.24 or higher (for local development)
- PostgreSQL (if not using Docker)
- Redis (if not using Docker)

### Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/fairuzald/library-system.git
   cd library-system
   ```

2. Set up environment variables:

   ```bash
   cp .env.example.dev .env.dev
   ```

3. Start the services with Docker Compose:

   ```bash
   docker-compose -f deployment/docker/docker-compose.dev.yml up

   # or using make

   make dev
   ```

4. The services will be available at:
   - API Gateway: http://localhost:8000
   - Book Service: http://localhost:8080
   - Category Service: http://localhost:8081
   - User Service: http://localhost:8082

### Local Development

For development without Docker:

1. Install dependencies:

   ```bash
   go mod download
   ```

2. Set up databases:

   ```bash
   # Create PostgreSQL databases: book_db, category_db, user_db
   # Run migrations
   dbmate -d migrations/book up
   dbmate -d migrations/category up
   dbmate -d migrations/user up
   ```

### Authentication

All protected endpoints require a valid JWT token in the Authorization header:

```
Authorization: Bearer <your_token>
```

To get a token:

1. Register a user: `POST /api/auth/register`
2. Login: `POST /api/auth/login`
3. Use the returned token in subsequent requests

## Entity Relationship Diagram

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

## Deployment

The project can be deployed to any cloud platform that supports Docker containers:

1. Build the Docker images:

   ```bash
   docker-compose -f deployment/docker/docker-compose.yml build
   ```

2. Push the images to Docker Hub:

   ```bash
   docker-compose -f deployment/docker/docker-compose.yml push
   ```

3. Deploy on your preferred cloud platform using the Docker images.

## Limitations and Known Issues

- The project doesn't include a frontend application
- Rate limiting is basic and might need adjustment for production
- Automated testing could be more comprehensive
