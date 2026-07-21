CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL CHECK (btrim(service_name) <> ''),
    price BIGINT NOT NULL CHECK (price > 0),
    user_id UUID NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subscriptions_valid_period CHECK (end_date IS NULL OR end_date >= start_date),
    CONSTRAINT subscriptions_start_month CHECK (start_date = date_trunc('month', start_date)::date),
    CONSTRAINT subscriptions_end_month CHECK (end_date IS NULL OR end_date = date_trunc('month', end_date)::date)
);

CREATE INDEX IF NOT EXISTS subscriptions_user_id_idx
    ON subscriptions (user_id);

CREATE INDEX IF NOT EXISTS subscriptions_service_name_idx
    ON subscriptions (service_name);

CREATE INDEX IF NOT EXISTS subscriptions_period_idx
    ON subscriptions (start_date, end_date);
