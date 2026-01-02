# Integration Tests

This directory contains integration tests for the ModelMatrix REST API.

## Structure

```
tests/
├── integration/          # Integration tests (full HTTP requests)
│   ├── setup.go         # Test setup/teardown helpers
│   ├── auth_test.go     # Authentication tests
│   ├── collections_test.go
│   ├── datasources_test.go
│   └── ...
├── testdata/             # Test data files (Go convention - ignored by builds)
│   └── hmeq.csv         # Sample CSV file for testing
└── helpers/              # Test helper utilities
```

## Running Tests

### Prerequisites

1. **Test Database**: A separate test PostgreSQL database (or use testcontainers)
2. **Test MinIO**: A test MinIO instance (or use testcontainers)
3. **Test LDAP**: A test LDAP server (or use testcontainers)

### Environment Variables

The test setup automatically loads environment variables from `tests/integration/.env.test` if it exists. You can:

1. **Use the provided `.env.test` file** (already created):
   - The file is automatically loaded when you run tests
   - Edit `tests/integration/.env.test` to customize values for your environment

2. **Set environment variables manually** (overrides `.env.test`):
   ```bash
   export TEST_DB_HOST=localhost
   export TEST_DB_PORT=5432
   export TEST_DB_USER=postgres
   export TEST_DB_PASSWORD=dayang
   export TEST_DB_NAME=modelmatrixtest
   
   export TEST_MINIO_ENDPOINT=localhost:9000
   export TEST_MINIO_ACCESS_KEY=minioadmin
   export TEST_MINIO_SECRET_KEY=minioadmin123
   export TEST_MINIO_BUCKET=modelmatrixtest
   
   export TEST_LDAP_HOST=localhost
   export TEST_LDAP_PORT=3890
   export TEST_LDAP_BASE_DN=dc=example,dc=org
   export TEST_LDAP_BIND_DN=cn=admin,dc=example,dc=org
   export TEST_LDAP_BIND_PASSWORD=admin
   
   export TEST_JWT_SECRET=test-secret-key-change-in-production
   export TEST_LOG_LEVEL=info
   ```

### Run All Integration Tests

```bash
make test-integration
```

### Run Specific Test

```bash
go test -v ./tests/integration -run TestCreateCollection
```

### Run with Coverage

```bash
make test-integration-coverage
```

## Test Approach

1. **Full HTTP Stack**: Tests make actual HTTP requests to a test server
2. **Real Dependencies**: Uses real PostgreSQL, MinIO, and LDAP (or testcontainers)
3. **Isolated**: Each test runs in a transaction that's rolled back, or uses a fresh database
4. **Test Data**: Test data files are stored in `tests/testdata/` (Go convention - ignored by builds)

## Writing New Tests

See `collections_test.go` for an example. Key patterns:

- Use `setupTestServer()` to get a test HTTP client
- Use `authenticate()` to get a JWT token
- Use `cleanup()` to clean up test data
- Use `testify` for assertions

