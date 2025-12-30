package kalshi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
)

const (
	defaultBaseURL   = "https://api.elections.kalshi.com/trade-api/v2/events"
	defaultSeriesURL = "https://api.elections.kalshi.com/trade-api/v2/series"
	defaultBookURL   = "https://api.elections.kalshi.com/trade-api/v2/markets"
)

// Client talks to the Kalshi Trade API.
type Client struct {
	baseURL    string
	seriesURL  string
	bookURL    string
	httpClient *http.Client
	nextCursor string
}

// Config provides optional overrides.
type Config struct {
	BaseURL   string
	SeriesURL string
	BookURL   string
	Timeout   time.Duration
}

// NewClient builds a configured Kalshi API client.
func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	series := cfg.SeriesURL
	if series == "" {
		series = defaultSeriesURL
	}
	book := cfg.BookURL
	if book == "" {
		book = defaultBookURL
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 20 * time.Second
	}
	return &Client{
		baseURL:   base,
		seriesURL: series,
		bookURL:   book,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Name() string {
	return "kalshi"
}

// Fetch retrieves a single page of open events and advances the internal cursor.
// When the end is reached, the cursor is reset to start over on the next call.
func (c *Client) Fetch(ctx context.Context, opts collectors.FetchOptions) ([]collectors.Event, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 100 // default fallback
	}
	if pageSize > 200 {
		pageSize = 200 // API limit
	}

	resp, err := c.listEvents(ctx, pageSize, c.nextCursor)
	if err != nil {
		return nil, fmt.Errorf("list kalshi events: %w", err)
	}

	log.Printf("[kalshi] processing batch of %d events (cursor: %s)", len(resp.Events), c.nextCursor)
	var events []collectors.Event
	for _, evt := range resp.Events {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		detail, err := c.fetchEvent(ctx, evt.Ticker)
		if err != nil {
			log.Printf("[kalshi] skip event %s: %v", evt.Ticker, err)
			continue
		}

		series, err := c.fetchSeries(ctx, evt.SeriesTicker)
		if err != nil {
			log.Printf("[kalshi] skip series %s for event %s: %v", evt.SeriesTicker, evt.Ticker, err)
			continue
		}

		events = append(events, c.normalizeEvent(ctx, detail, series))
	}

	c.nextCursor = resp.Cursor
	if c.nextCursor == "" {
		log.Printf("[kalshi] reached end of events, resetting cursor")
	}

	return events, nil
}

func (c *Client) listEvents(ctx context.Context, limit int, cursor string) (*eventsResponse, error) {
	u, _ := url.Parse(c.baseURL)
	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("status", "open")
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	var out eventsResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) fetchEvent(ctx context.Context, ticker string) (*eventDetail, error) {
	u := fmt.Sprintf("%s/%s?with_nested_markets=true", strings.TrimRight(c.baseURL, "/"), ticker)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out eventDetail
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) fetchSeries(ctx context.Context, ticker string) (*seriesResponse, error) {
	u := fmt.Sprintf("%s/%s", strings.TrimRight(c.seriesURL, "/"), ticker)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out seriesResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) fetchOrderbooks(ctx context.Context, ticker string) (map[string]collectors.Orderbook, error) {
	u := fmt.Sprintf("%s/%s/orderbook?depth=5", strings.TrimRight(c.bookURL, "/"), ticker)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out orderbookResponse
	if err := c.do(req, &out); err != nil {
		return nil, err
	}

	yesBids := convertLevels(out.Yes)
	noBids := convertLevels(out.No)

	yesBook := collectors.Orderbook{
		Bids: yesBids,
		Asks: deriveAsksFromOpposite(noBids),
	}
	noBook := collectors.Orderbook{
		Bids: noBids,
		Asks: deriveAsksFromOpposite(yesBids),
	}

	return map[string]collectors.Orderbook{
		"yes": yesBook,
		"no":  noBook,
	}, nil
}

func (c *Client) do(req *http.Request, dst any) error {
	var attempt int
	for {
		attempt++
		resp, err := c.httpClient.Do(req)
		if err != nil {
			if shouldRetry(attempt, 0) {
				sleep(attempt)
				continue
			}
			return err
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer resp.Body.Close()
			return json.NewDecoder(resp.Body).Decode(dst)
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()

		if shouldRetry(attempt, resp.StatusCode) {
			sleep(attempt)
			continue
		}
		return fmt.Errorf("kalshi API %s: %s", resp.Status, string(body))
	}
}

func (c *Client) normalizeEvent(ctx context.Context, detail *eventDetail, series *seriesResponse) collectors.Event {
	ev := detail.Event

	var closeTime time.Time
	if ev.CloseTime != "" {
		if ts, err := time.Parse(time.RFC3339, ev.CloseTime); err == nil {
			closeTime = ts
		}
	}

	var settlement []collectors.ResolutionSource
	if series != nil {
		for _, s := range series.Series.SettlementSources {
			settlement = append(settlement, collectors.ResolutionSource{
				Name: s.Name,
				URL:  s.URL,
			})
		}
	}

	norm := collectors.Event{
		Venue:             collectors.VenueKalshi,
		EventID:           ev.Ticker,
		Title:             ev.Title,
		Description:       ev.Description,
		Category:          ev.Category,
		Status:            ev.Status,
		ResolutionSource:  strings.Join(ev.ResolutionSources, ", "),
		ResolutionDetails: strings.TrimSpace(ev.RulesPrimary + "\n" + ev.RulesSecondary),
		SettlementSources: settlement,
		CloseTime:         closeTime,
		ContractTermsURL:  series.Series.ContractTermsURL,
		Raw:               map[string]any{"raw_event": detail},
	}

	markets := detail.Markets
	if len(markets) == 0 {
		markets = ev.Markets
	}
	for _, m := range markets {
		if m.Status != "active" {
			continue
		}
		norm.Markets = append(norm.Markets, c.normalizeMarket(ctx, ev, &m, series))
	}
	return norm
}

func (c *Client) normalizeMarket(ctx context.Context, ev event, m *market, series *seriesResponse) collectors.Market {
	var closeTime time.Time
	if m.CloseTime != "" {
		if ts, err := time.Parse(time.RFC3339, m.CloseTime); err == nil {
			closeTime = ts
		}
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	orderbooks := make(map[string]collectors.Orderbook)
	if books, err := c.fetchOrderbooks(ctxWithTimeout, m.Ticker); err == nil {
		orderbooks = books
	}

	refURL := ""
	if series != nil {
		refURL = series.Series.ContractURL
	}

	displayQuestion := deriveKalshiQuestion(ev.Title, m)

	return collectors.Market{
		MarketID:     m.Ticker,
		Question:     displayQuestion,
		Subtitle:     m.SubTitle,
		TickSize:     float64(m.TickSize) / 100.0,
		CloseTime:    closeTime,
		Volume:       float64(m.Volume),
		Volume24h:    float64(m.Volume24h),
		OpenInterest: float64(m.OpenInterest),
		Price: collectors.PriceSnapshot{
			YesBid: centsToFloat(m.YesBid),
			YesAsk: centsToFloat(m.YesAsk),
			NoBid:  centsToFloat(m.NoBid),
			NoAsk:  centsToFloat(m.NoAsk),
		},
		Orderbooks:   orderbooks,
		ReferenceURL: refURL,
	}
}

func centsToFloat(v int64) float64 {
	return float64(v) / 100.0
}

func convertLevels(levels [][]int64) []collectors.OrderbookLevel {
	out := make([]collectors.OrderbookLevel, 0, len(levels))
	for _, lvl := range levels {
		if len(lvl) < 2 {
			continue
		}
		price := centsToFloat(lvl[0])
		qty := float64(lvl[1])
		out = append(out, collectors.OrderbookLevel{
			Price:     price,
			Quantity:  qty,
			RawPrice:  float64(lvl[0]),
			RawAmount: qty,
		})
	}
	return out
}

func deriveAsksFromOpposite(oppositeBids []collectors.OrderbookLevel) []collectors.OrderbookLevel {
	if len(oppositeBids) == 0 {
		return nil
	}
	asks := make([]collectors.OrderbookLevel, 0, len(oppositeBids))
	for _, lvl := range oppositeBids {
		price := 1 - lvl.Price
		if price < 0 {
			price = 0
		}
		if price > 1 {
			price = 1
		}
		asks = append(asks, collectors.OrderbookLevel{
			Price:     price,
			Quantity:  lvl.Quantity,
			RawPrice:  100 - lvl.RawPrice,
			RawAmount: lvl.RawAmount,
		})
	}
	return asks
}

type eventsResponse struct {
	Events []event `json:"events"`
	Cursor string  `json:"cursor"`
}

type event struct {
	Ticker            string   `json:"event_ticker"`
	SeriesTicker      string   `json:"series_ticker"`
	Title             string   `json:"title"`
	SubTitle          string   `json:"sub_title"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	Category          string   `json:"category"`
	CloseTime         string   `json:"close_time"`
	ResolutionSources []string `json:"settlement_sources"`
	RulesPrimary      string   `json:"rules_primary"`
	RulesSecondary    string   `json:"rules_secondary"`
	Markets           []market `json:"markets"`
}

type eventDetail struct {
	Event   event    `json:"event"`
	Markets []market `json:"markets"`
}

type market struct {
	Ticker         string `json:"ticker"`
	Title          string `json:"title"`
	SubTitle       string `json:"sub_title"`
	Status         string `json:"status"`
	Result         string `json:"result"`
	YesAsk         int64  `json:"yes_ask"`
	YesBid         int64  `json:"yes_bid"`
	NoAsk          int64  `json:"no_ask"`
	NoBid          int64  `json:"no_bid"`
	Volume         int64  `json:"volume"`
	Volume24h      int64  `json:"volume_24h"`
	OpenInterest   int64  `json:"open_interest"`
	RulesPrimary   string `json:"rules_primary"`
	RulesSecondary string `json:"rules_secondary"`
	CloseTime      string `json:"close_time"`
	TickSize       int64  `json:"tick_size"`
	ContractURL    string `json:"contract_url"`
}

type orderbookResponse struct {
	Yes [][]int64 `json:"yes"`
	No  [][]int64 `json:"no"`
}

type seriesResponse struct {
	Series series `json:"series"`
}

type series struct {
	Ticker            string             `json:"ticker"`
	SettlementSources []settlementSource `json:"settlement_sources"`
	ContractTermsURL  string             `json:"contract_terms_url"`
	ContractURL       string             `json:"contract_url"`
}

type settlementSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func shouldRetry(attempt int, status int) bool {
	if attempt >= 5 {
		return false
	}
	if status == 0 {
		return true
	}
	if status == http.StatusTooManyRequests || status >= 500 {
		return true
	}
	return false
}

func deriveKalshiQuestion(eventTitle string, m *market) string {
	base := m.Title

	alias := extractEntityFromRules(m.RulesPrimary)
	if alias == "" {
		alias = extractEntityFromTitle(eventTitle)
	}
	if alias == "" && strings.Contains(base, "  ") {
		alias = extractEntityFromTitle(base)
	}
	if alias == "" {
		if parts := strings.Split(m.Ticker, "-"); len(parts) > 0 {
			alias = parts[len(parts)-1]
		}
	}

	if alias == "" {
		return base
	}

	// Already present
	if strings.Contains(strings.ToLower(base), strings.ToLower(alias)) {
		return base
	}

	// Replace double-space placeholder ("Will  become ...")
	if strings.Contains(base, "  ") {
		return strings.Replace(base, "  ", " "+alias+" ", 1)
	}

	return fmt.Sprintf("%s (%s)", base, alias)
}

func extractEntityFromRules(rule string) string {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return ""
	}
	lower := strings.ToLower(rule)
	if !strings.HasPrefix(lower, "if ") {
		return ""
	}
	trimmed := strings.TrimSpace(rule[3:])
	lowerTrimmed := strings.ToLower(trimmed)
	keywords := []string{" becomes", " is ", " wins", " will ", " reaches", " secures", " scores", " resigns", " retires", " defeats", " beats", " finishes", " captures", " takes", " makes", " receives", " gets "}
	pos := -1
	for _, kw := range keywords {
		if idx := strings.Index(lowerTrimmed, kw); idx != -1 && (pos == -1 || idx < pos) {
			pos = idx
		}
	}
	if pos == -1 {
		if idx := strings.Index(lowerTrimmed, ","); idx != -1 {
			pos = idx
		} else if idx := strings.Index(lowerTrimmed, " then"); idx != -1 {
			pos = idx
		} else {
			pos = len(trimmed)
		}
	}
	alias := strings.TrimSpace(trimmed[:pos])
	alias = strings.Trim(alias, `"'`)
	return alias
}

func extractEntityFromTitle(title string) string {
	title = strings.TrimSpace(title)
	lower := strings.ToLower(title)
	if !strings.HasPrefix(lower, "will ") {
		return ""
	}
	title = title[5:] // drop "Will "
	lower = strings.ToLower(title)

	endIdx := strings.Index(lower, " become")
	if endIdx == -1 {
		endIdx = strings.Index(lower, " be ")
	}
	if endIdx == -1 {
		return ""
	}
	alias := strings.TrimSpace(title[:endIdx])
	alias = strings.Trim(alias, `"'`)
	return alias
}

func sleep(attempt int) {
	backoff := time.Duration(1<<uint(attempt-1)) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	time.Sleep(backoff)
}
