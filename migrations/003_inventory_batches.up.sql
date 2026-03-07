CREATE TABLE inventory_batches (
    id UUID PRIMARY KEY,
    sku TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity >= 0),
    purchase_price DECIMAL(10, 2) NOT NULL,
    expiry_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE inventory_waste (
    id UUID PRIMARY KEY,
    batch_id UUID REFERENCES inventory_batches(id),
    quantity INTEGER NOT NULL,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
