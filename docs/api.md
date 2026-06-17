# API Reference

Interactive documentation is served at:

```text
http://localhost:8080/swagger/index.html
```

The machine-readable contract is [`openapi.yaml`](openapi.yaml).

## Authentication

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/auth/login` | Public |
| POST | `/api/auth/register` | Public |

Protected routes require `Authorization: Bearer <JWT>`.

## Customers

| Method | Path |
|--------|------|
| GET | `/api/customers` |
| POST | `/api/customers` |
| GET | `/api/customers/{id}` |
| PUT | `/api/customers/{id}` |
| PATCH | `/api/customers/{id}/deactivate` |

## Products

| Method | Path |
|--------|------|
| GET | `/api/products` |
| POST | `/api/products` |
| GET | `/api/products/{id}` |
| PUT | `/api/products/{id}` |
| PATCH | `/api/products/{id}/deactivate` |

## Orders

| Method | Path |
|--------|------|
| GET | `/api/orders` |
| POST | `/api/orders` |
| GET | `/api/orders/{id}` |
| PATCH | `/api/orders/{id}/cancel` |
| PATCH | `/api/orders/{id}/ship` |

## Infrastructure

| Method | Path | Auth |
|--------|------|------|
| GET | `/health` | Public |
| GET | `/metrics` | Public |
| GET | `/swagger/index.html` | Public |

## Error responses

All error responses follow architecture §24:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "invalid request body"
  }
}
```

Common codes: `VALIDATION_ERROR`, `UNAUTHORIZED`, `FORBIDDEN`, `RESOURCE_NOT_FOUND`, `DUPLICATE_RESOURCE`, `INVALID_ORDER_STATUS`, `INACTIVE_CUSTOMER`, `INACTIVE_PRODUCT`, `INTERNAL_ERROR`.

## Pagination and filtering

List endpoints accept `page`, `page_size`, and resource-specific filters. See OpenAPI parameters for details.
