CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(50) PRIMARY KEY,
    customer_id VARCHAR(50) NOT NULL,
    item_name VARCHAR(100) NOT NULL,
    amount BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- Function that fires a pg_notify event whenever an order's status changes.
-- Payload is a JSON object: {"order_id": "...", "status": "...", "updated_at": "..."}
CREATE OR REPLACE FUNCTION notify_order_status_change()
RETURNS TRIGGER AS $$
DECLARE
    payload JSON;
BEGIN
    IF NEW.status IS DISTINCT FROM OLD.status THEN
        payload := json_build_object(
            'order_id',   NEW.id,
            'status',     NEW.status,
            'updated_at', to_char(NOW() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
        );
        PERFORM pg_notify('order_status_updates', payload::text);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop the old trigger if it exists, then recreate it.
DROP TRIGGER IF EXISTS order_status_change_trigger ON orders;

CREATE TRIGGER order_status_change_trigger
    AFTER UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION notify_order_status_change();
