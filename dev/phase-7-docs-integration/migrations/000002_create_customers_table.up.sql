CREATE TABLE customers (
    id UUID PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    document VARCHAR(30) NOT NULL UNIQUE,
    email VARCHAR(160),
    phone VARCHAR(30),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_customers_document ON customers(document);
