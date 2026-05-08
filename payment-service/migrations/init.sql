CREATE TABLE IF NOT EXISTS payments (
    id VARCHAR(50) PRIMARY KEY,
    order_id VARCHAR(50) NOT NULL UNIQUE,
    customer_email VARCHAR(255) NOT NULL,
    transaction_id VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL
);
