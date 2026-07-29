ALTER TABLE order_modifications
ADD COLUMN IF NOT EXISTS backorder_id INTEGER REFERENCES backorders(id);
