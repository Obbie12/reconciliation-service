# Auto-Reconciliation Service

A high-performance financial reconciliation system that matches and synchronizes data between bank statements and internal accounting records. The service handles both one-to-one and one-to-many relationships while maintaining ACID compliance.

## Features

- Automated reconciliation of financial transactions
- Support for one-to-one and one-to-many relationships
- High-performance matching engine (10,000+ records within 30 seconds)
- ACID compliant database operations
- Comprehensive audit trail
- RESTful API interface
- Configurable matching rules
- Detailed reporting and status tracking

## Technology Stack

- Language: Go
- Database: MySQL
- Optional: Redis for caching

## Project Structure

```
reconciliation-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── handlers/
│   ├── models/
│   ├── repositories/
│   ├── services
│   └── matching/ 
├── migrations/
├── tests/
└── README.md
```

## Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/Obbie12/reconciliation-service.git
   cd reconciliation-service
   ```

2. Set up the environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. Install dependencies:
   ```bash
   go mod tidy
   ```

4. Environment Setup:
   ```bash
   # Create .env file from template
   make env-setup

   # Edit .env file with your configuration
   vim .env
   ```

   The service will automatically create the database if it doesn't exist, using the credentials provided in the environment variables.

   Required environment variables in .env:
   ```env
   # Server Configuration
   SERVER_ADDRESS=:8080
   ENVIRONMENT=development

   # Database Configuration
   DB_HOST=localhost
   DB_PORT=3306
   DB_USER=root
   DB_PASSWORD=your_password_here
   DB_NAME=reconciliation_db
   DB_PARAMS=parseTime=true

   # Migration Configuration
   MIGRATION_DIR=migrations
   ```

5. Quick Setup:
   ```bash
   # Install dependencies, build, and run migrations
   make setup

   # Run the service
   make run
   ```

   Available commands:
   ```bash
   # Build and run
   make build
   make run

   # Database migrations
   make migrate-up
   make migrate-down
   make migrate-version

   # Development
   make test
   make clean
   make deps
   make setup
   ```

   You can also run migrations using the service binary directly:
   ```bash
   # Run migrations
   ./reconciliation-service -migrate=up
   ./reconciliation-service -migrate=down
   ./reconciliation-service -migrate=version

   # Run migrations with specific steps
   ./reconciliation-service -migrate=up -steps=1
   ./reconciliation-service -migrate=down -steps=1
   ```

## API Endpoints

### Reconciliation Endpoints

#### Start Reconciliation
```http
POST /api/v1/reconciliation/start
{
    "from_date": "2024-01-01",
    "to_date": "2024-01-31"
}
```

#### Get Reconciliation Status
```http
GET /api/v1/reconciliation/{batch_id}/status
```

#### Resolve Dispute
```http
POST /api/v1/reconciliation/{batch_id}/resolve
{
    "resolution": "matched",
    "notes": "Manually verified"
}
```

#### Get Unmatched Records
```http
GET /api/v1/reconciliation/unmatched?from_date=2024-01-01&to_date=2024-01-31
```

### Data Endpoints

#### insert Bank Transactions
```http
POST /api/v1/data/bank-transactions
[
    {
        "transaction_id": "BNK001",
        "account_number": "1234567890",
        "amount": 1500.00,
        "transaction_date": "2024-01-15",
        "description": "Payment received",
        "reference_number": "INV123"
    },
    {
        "transaction_id": "BNK002",
        "account_number": "1234567890",
        "amount": 1000.00,
        "transaction_date": "2024-01-15",
        "description": "Payment received",
        "reference_number": "INV124"
    },
    {
        "transaction_id": "BNK003",
        "account_number": "1234567890",
        "amount": 900.00,
        "transaction_date": "2024-01-15",
        "description": "Payment received",
        "reference_number": "INV125"
    },
    {
        "transaction_id": "BNK004",
        "account_number": "1234567890",
        "amount": 800.00,
        "transaction_date": "2024-01-15",
        "description": "Payment received",
        "reference_number": "INV126"
    }
]
```

#### Ingest Accounting Entries
```http
POST /api/v1/data/accounting-entries
[
    {
        "entry_id": "ACC001",
        "account_code": "AR001",
        "amount": 1500.00,
        "entry_date": "2024-01-15",
        "description": "Invoice payment",
        "invoice_number": "INV123"
    },
    {
        "entry_id": "ACC002",
        "account_code": "AR001",
        "amount": 700.00,
        "entry_date": "2024-01-15",
        "description": "Invoice payment",
        "invoice_number": "INV124"
    },
    {
        "entry_id": "ACC003",
        "account_code": "AR001",
        "amount": 300.00,
        "entry_date": "2024-01-15",
        "description": "Invoice payment",
        "invoice_number": "INV124"
    },
    {
        "entry_id": "ACC004",
        "account_code": "AR001",
        "amount": 1000.00,
        "entry_date": "2024-01-15",
        "description": "Invoice payment",
        "invoice_number": "INV125"
    },
    {
        "entry_id": "ACC005",
        "account_code": "AR001",
        "amount": 900.00,
        "entry_date": "2024-01-15",
        "description": "Invoice payment",
        "invoice_number": "INV126"
    }
]
```

## Configuration

The service can be configured using environment variables:

```env
# Server Configuration
SERVER_ADDRESS=:8080
ENVIRONMENT=development

# Database Configuration
MYSQL_DSN=user:password@tcp(localhost:3306)/reconciliation_db?parseTime=true
MYSQL_MAX_OPEN_CONNS=25
MYSQL_MAX_IDLE_CONNS=25

# Redis Configuration (Optional)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Matching Engine Configuration
MATCH_CONFIDENCE_THRESHOLD=0.8
DATE_TOLERANCE_DAYS=3
AMOUNT_TOLERANCE_PERCENT=0.01
```

## Performance Optimization

The service is optimized for high performance:

- Efficient database indexing
- Connection pooling
- Batch processing
- Optional Redis caching
- Optimized matching algorithms

## Testing

Run the test suite:

```bash
go test ./...
```

Run with coverage:

```bash
go test -cover ./...
```

## Monitoring and Metrics

The service exposes a health check endpoint:

```http
GET /health
```

## Error Handling

The service uses standard HTTP status codes:

- 200: Success
- 206: Partial Content (some records failed)
- 400: Bad Request
- 404: Not Found
- 500: Internal Server Error

Error responses include detailed messages:

```json
{
    "error": "Detailed error message",
    "details": {
        "failed_records": ["id1", "id2"],
        "reason": "Validation failed"
    }
}
```
