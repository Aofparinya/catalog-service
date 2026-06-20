# Order Platform Catalog Service

Go service for categories, products, SKUs, prices, warehouses, inventory, stock
movements, and stock reservations.

## Local development

```powershell
copy .env.example .env
go mod download
go run ./cmd/server
```

The service listens on `http://localhost:3003` and exposes APIs under
`/api/v1`. Docker PostgreSQL is available at `127.0.0.1:15432`.

## Validation

```powershell
go fmt ./...
go vet ./...
go test ./...
```
