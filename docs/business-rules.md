# Business Rules

Canonical rules from architecture §14.

## Customers

| ID | Rule |
|----|------|
| BR-CUS-001 | Customer name is required. |
| BR-CUS-002 | Customer document must be unique. |
| BR-CUS-003 | Inactive customers cannot create new orders. |
| BR-CUS-004 | Customers with orders must not be physically deleted. |

## Products

| ID | Rule |
|----|------|
| BR-PRD-001 | Product name is required. |
| BR-PRD-002 | Product SKU must be unique. |
| BR-PRD-003 | Product price must be greater than zero. |
| BR-PRD-004 | Inactive products cannot be used in new orders. |
| BR-PRD-005 | Product price must be copied into the order item at order creation time. |

## Orders

| ID | Rule |
|----|------|
| BR-ORD-001 | Order must have a customer. |
| BR-ORD-002 | Order must have at least one item. |
| BR-ORD-003 | Item quantity must be greater than zero. |
| BR-ORD-004 | Order total is the sum of all item totals. |
| BR-ORD-005 | Order is created with status CREATED. |
| BR-ORD-006 | Created order must be sent to the processing queue. |
| BR-ORD-007 | Only CREATED orders can be canceled. |
| BR-ORD-008 | Only PAID orders can be shipped. |
| BR-ORD-009 | Worker can process only CREATED orders. |
| BR-ORD-010 | Failed processing must move the order to FAILED. |

## Authentication and Authorization

| ID | Rule |
|----|------|
| BR-AUTH-001 | Passwords are stored hashed, never plain text. |
| BR-AUTH-002 | Inactive users cannot log in. |
| BR-AUTH-003 | JWT is required for protected endpoints. |
| BR-AUTH-004 | Role authorization follows the matrix in architecture §15. |
