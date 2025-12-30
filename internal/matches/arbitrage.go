package matches

type Direction string

const (
	DirectionNone                Direction = ""
	DirectionBuyYesPMBuyNoKalshi Direction = "BUY_YES_PM_BUY_NO_KALSHI"
	DirectionBuyNoPMBuyYesKalshi Direction = "BUY_NO_PM_BUY_YES_KALSHI"
)

type Leg struct {
	Venue    string  `json:"venue"`
	Side     string  `json:"side"`
	Outcome  string  `json:"outcome"`
	AvgPrice float64 `json:"avg_price"`
	Quantity float64 `json:"quantity"`
	CostUSD  float64 `json:"cost_usd"`
}

type Opportunity struct {
	Direction         Direction `json:"direction"`
	Quantity          float64   `json:"quantity"`
	ProfitUSD         float64   `json:"profit_usd"`
	TotalCostUSD      float64   `json:"total_cost_usd"`
	BudgetUSD         float64   `json:"budget_usd"`
	KalshiFeesUSD     float64   `json:"kalshi_fees_usd"`
	PolymarketFeesUSD float64   `json:"polymarket_fees_usd"`
	Legs              []Leg     `json:"legs"`
}
