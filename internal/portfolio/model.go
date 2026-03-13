package portfolio

import (
	"encoding/json"
	"time"
)

type AssetType string

const (
	AssetTypeStock      AssetType = "stock"
	AssetTypeFD         AssetType = "fd"
	AssetTypeMutualFund AssetType = "mutual_fund"
	AssetTypeGold       AssetType = "gold"
	AssetTypeOther      AssetType = "other"
)

// Holding is the HTTP-facing representation of a portfolio holding.
type Holding struct {
	ID           int             `json:"id"`
	AssetType    AssetType       `json:"asset_type"`
	Name         string          `json:"name"`
	Quantity     *float64        `json:"quantity"`
	BuyPrice     *float64        `json:"buy_price"`
	CurrentValue *float64        `json:"current_value"`
	BuyDate      *time.Time      `json:"buy_date"`
	Notes        string          `json:"notes"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CreateHoldingRequest struct {
	AssetType    AssetType       `json:"asset_type"`
	Name         string          `json:"name"`
	Quantity     *float64        `json:"quantity"`
	BuyPrice     *float64        `json:"buy_price"`
	CurrentValue *float64        `json:"current_value"`
	BuyDate      *time.Time      `json:"buy_date"`
	Notes        string          `json:"notes"`
	Metadata     json.RawMessage `json:"metadata"`
}

type UpdateHoldingRequest struct {
	AssetType    AssetType       `json:"asset_type"`
	Name         string          `json:"name"`
	Quantity     *float64        `json:"quantity"`
	BuyPrice     *float64        `json:"buy_price"`
	CurrentValue *float64        `json:"current_value"`
	BuyDate      *time.Time      `json:"buy_date"`
	Notes        string          `json:"notes"`
	Metadata     json.RawMessage `json:"metadata"`
}
