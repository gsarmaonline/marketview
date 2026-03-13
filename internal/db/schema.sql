CREATE TABLE IF NOT EXISTS supply_chain_store (
    symbol      VARCHAR(50) NOT NULL,
    report_year VARCHAR(20) NOT NULL,
    entities    JSONB       NOT NULL,
    financials  JSONB,
    parsed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (symbol, report_year)
);

ALTER TABLE supply_chain_store ADD COLUMN IF NOT EXISTS financials JSONB;

CREATE TABLE IF NOT EXISTS shareholding_pattern_store (
    symbol          VARCHAR(50)  NOT NULL,
    quarter_end     VARCHAR(30)  NOT NULL,
    pattern         JSONB        NOT NULL,
    fetched_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (symbol, quarter_end)
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
