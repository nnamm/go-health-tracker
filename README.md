# Go-SQLite3 Health Tracker CRUD lab

A simple REST API for tracking health-related metrics. It uses an SQLite3 database to store health data and is implemented in Go.

## Purpose

This project aims to build a system for tracking health-related data. It currently supports step count tracking, with plans to add other health metrics such as sleep duration in the future.

## API Endpoints

### Health Record Management

**Endpoint**: `/health/records`

| Method | Parameters         | Description                                                  |
| ------ | ------------------ | ------------------------------------------------------------ |
| GET    | date=YYYYMMDD      | Retrieve a health record for a specific date                 |
| GET    | year=YYYY&month=MM | Retrieve health records for the specified year and month     |
| GET    | year=YYYY          | Retrieve all health records for the specified year           |
| POST   | -                  | Create a new health record (JSON data in request body)       |
| PUT    | -                  | Update an existing health record (JSON data in request body) |
| DELETE | date=YYYYMMDD      | Delete a health record for the specified date                |

## Request/Response Examples

### Create a Health Record (POST)

```bash
curl -X POST http://localhost:8000/health/records \
  -H "Content-Type: application/json" \
  -d '{"date":"2024-05-01","step_count":12345}'
```

Response:

```json
{
  "records": [
    {
      "id": 1,
      "date": "2025-05-01",
      "step_count": 12345,
      "created_at": "2025-05-06T10:30:00Z",
      "updated_at": "2025-05-06T10:30:00Z"
    }
  ]
}
```

### Retrieve a Health Record (GET)

```bash
curl -X GET "http://localhost:8000/health/records?date=20240501"
```

Response:

```json
{
  "records": [
    {
      "id": 1,
      "date": "2024-05-01",
      "step_count": 12345,
      "created_at": "2024-05-06T10:30:00Z",
      "updated_at": "2024-05-06T10:30:00Z"
    }
  ]
}
```

## Project Structure

```
.
├── cmd
│   └── server
│       ├── main.go      - Server startup and routing configuration
│       └── main_test.go - Integration tests
└── internal
    ├── apperr           - Application error definitions
    ├── database         - Database operations
    ├── handlers         - HTTP request handlers
    ├── models           - Data models
    └── validators       - Data validation
```

## Development Plan

- Phase 1: Backend foundation and API implementation (Completed)
- Phase 2: Migration to Gin framework, addition of authentication, migration to PostgreSQL (Planned)

## How to Run

```bash
# Start the server
go run cmd/server/main.go

# Run tests
go test ./...
```

## License

MIT
