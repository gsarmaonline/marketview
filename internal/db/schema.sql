CREATE TABLE IF NOT EXISTS supply_chain_store (
    symbol      VARCHAR(50) NOT NULL,
    report_year VARCHAR(20) NOT NULL,
    entities    JSONB       NOT NULL,
    parsed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (symbol, report_year)
);

CREATE TABLE IF NOT EXISTS holdings (
    id            SERIAL PRIMARY KEY,
    asset_type    VARCHAR(50)   NOT NULL,
    name          VARCHAR(255)  NOT NULL,
    quantity      NUMERIC(15,4),
    buy_price     NUMERIC(15,2),
    current_value NUMERIC(15,2),
    buy_date      DATE,
    notes         TEXT          NOT NULL DEFAULT '',
    metadata      JSONB         NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
