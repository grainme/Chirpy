# Chirpy

Chirpy allows users to create short text posts ("chirps"), manage accounts, and interact with a clean RESTful API.

## Features

- **User Management**: Registration, login, and profile updates with JWT authentication
- **Chirp Posts**: Create, read, and delete short messages (max 140 characters)
- **Profanity Filtering**: Automatic content moderation for chirps
- **JWT Authentication**: Secure token-based authentication with refresh tokens
- **Premium Memberships**: Chirpy Red subscription support via webhooks
- **PostgreSQL Database**: Type-safe queries using SQLc
- **Database Migrations**: Version-controlled schema changes with Goose

## Tech Stack

- **Language**: Go 1.24.5
- **Database**: PostgreSQL
- **Authentication**: JWT tokens with Argon2id password hashing
- **Code Generation**: SQLc for type-safe database queries
- **Key Libraries**:
  - `github.com/golang-jwt/jwt/v5` - JWT implementation
  - `github.com/alexedwards/argon2id` - Password hashing
  - `github.com/lib/pq` - PostgreSQL driver
  - `github.com/google/uuid` - UUID generation


## Getting Started

### Prerequisites

- Go 1.24.5 or higher
- PostgreSQL
- [Goose](https://github.com/pressly/goose) for migrations
- [SQLc](https://sqlc.dev/) for code generation (optional, code is pre-generated)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/grainme/Chirpy.git
cd Chirpy
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables (create a `.env` file):
```env
DB_URL=postgres://username:password@localhost:5432/chirpy?sslmode=disable
JWT_SecretToken=your-secret-key-here
PLATFORM=dev
POLKA_KEY=your-polka-api-key
```

4. Run database migrations:
```bash
make migrate-up
```

5. Start the server:
```bash
go run main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Health & Admin

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/healthz` | Health check |
| GET | `/admin/metrics` | View metrics |
| POST | `/admin/reset` | Reset database (dev only) |

### Users

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/users` | No | Create new user |
| PUT | `/api/users` | JWT | Update user email/password |
| POST | `/api/login` | No | Login and receive JWT + refresh token |
| POST | `/api/refresh` | Refresh Token | Get new JWT token |
| POST | `/api/revoke` | Refresh Token | Revoke refresh token |

### Chirps

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/chirps` | JWT | Create a new chirp |
| GET | `/api/chirps` | No | Get all chirps (supports ?author_id=UUID&sort=desc/asc) |
| GET | `/api/chirps/{chirpID}` | No | Get specific chirp |
| DELETE | `/api/chirps/{chirpID}` | JWT | Delete chirp (owner only) |

### Webhooks

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/polka/webhooks` | API Key | Upgrade user to Chirpy Red |


## Features in Detail

### Profanity Filtering

Chirps are automatically filtered for profanity. The following words are replaced with asterisks:
- kerfuffle
- sharbert
- fornax

### Chirp Constraints

- Maximum body length: 140 characters
- Users can only delete their own chirps
- Chirps are linked to users via foreign key with cascade delete

### Premium Memberships

Users can be upgraded to "Chirpy Red" premium status via the Polka webhook integration.

## FYI

Built as part of learning Go and RESTful API design.
