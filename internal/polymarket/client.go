package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
)

const (
	defaultBaseURL = "https://gamma-api.polymarket.com/events"
	defaultBookURL = "https://clob.polymarket.com/book"
)

// Client fetches Polymarket events + CLOB data.
type Client struct {
	baseURL    string
	bookURL    string
	httpClient *http.Client
	nextOffset int
}

// Config controls optional overrides for the client.
type Config struct {
	BaseURL string
	BookURL string
	Timeout time.Duration
}

// NewClient builds a Polymarket client with sane defaults.
func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
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
		baseURL: base,
		bookURL: book,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Name() string {
	return "polymarket"
}

// Fetch retrieves a single page of open events and advances the internal offset.
// When the end of results is reached, the offset is reset to start over.
func (c *Client) Fetch(ctx context.Context, opts collectors.FetchOptions) ([]collectors.Event, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 50 // default fallback
	}

	list, err := c.listEvents(ctx, pageSize, c.nextOffset)
	if err != nil {
		return nil, fmt.Errorf("polymarket list events: %w", err)
	}
	if len(list) == 0 {
		log.Printf("[polymarket] reached end of events, resetting offset")
		c.nextOffset = 0
		return nil, nil
	}

	log.Printf("[polymarket] processing batch of %d summaries (offset: %d)", len(list), c.nextOffset)
	var events []collectors.Event
	for _, summary := range list {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if summary.Closed {
			continue
		}
		ev, err := c.fetchEvent(ctx, summary.ID)
		if err != nil {
			log.Printf("[polymarket] skip event %s: %v", summary.ID, err)
			continue
		}

		norm := c.normalizeEvent(ctx, ev)
		if len(norm.Markets) > 0 {
			events = append(events, norm)
		}
	}

	if len(list) < pageSize {
		log.Printf("[polymarket] reached end of events, resetting offset")
		c.nextOffset = 0
	} else {
		c.nextOffset += pageSize
	}

	return events, nil
}

func (c *Client) listEvents(ctx context.Context, limit, offset int) ([]eventSummary, error) {
	u, _ := url.Parse(c.baseURL)
	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	q.Set("closed", "false")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	var events []eventSummary
	if err := c.do(req, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (c *Client) fetchEvent(ctx context.Context, id string) (*eventDetail, error) {
	u := fmt.Sprintf("%s/%s", strings.TrimRight(c.baseURL, "/"), id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var ev eventDetail
	if err := c.do(req, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

func (c *Client) fetchOrderbook(ctx context.Context, tokenID string) (collectors.Orderbook, error) {
	u, _ := url.Parse(c.bookURL)
	q := u.Query()
	q.Set("token_id", tokenID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return collectors.Orderbook{}, err
	}

	var book clobBook
	if err := c.do(req, &book); err != nil {
		return collectors.Orderbook{}, err
	}
	return convertClobBook(book), nil
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
		return fmt.Errorf("polymarket API %s: %s", resp.Status, string(body))
	}
}

func (c *Client) normalizeEvent(ctx context.Context, ev *eventDetail) collectors.Event {
	var closeTime time.Time
	if ev.EndDate != "" {
		if ts, err := time.Parse(time.RFC3339, ev.EndDate); err == nil {
			closeTime = ts
		}
	}

	norm := collectors.Event{
		Venue:             collectors.VenuePolymarket,
		EventID:           ev.ID,
		Title:             ev.Title,
		Description:       ev.Description,
		Category:          ev.Category,
		Status:            map[bool]string{true: "closed", false: "open"}[ev.Closed],
		ResolutionSource:  ev.ResolutionSource,
		ResolutionDetails: ev.ResolutionDescription,
		CloseTime:         closeTime,
		Raw:               map[string]any{"raw_event": ev},
	}

	for _, m := range ev.Markets {
		if isPlaceholderMarket(&m) {
			continue
		}
		if m.Closed || !m.Active {
			continue
		}
		norm.Markets = append(norm.Markets, c.normalizeMarket(ctx, ev, &m))
	}
	return norm
}

func (c *Client) normalizeMarket(ctx context.Context, ev *eventDetail, m *market) collectors.Market {
	clobIDs := parseClobTokenIDs(m.ClobTokenIds)
	orderbooks := make(map[string]collectors.Orderbook)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for _, tokenID := range clobIDs {
		if tokenID == "" {
			continue
		}
		if book, err := c.fetchOrderbook(ctxWithTimeout, tokenID); err == nil {
			orderbooks[tokenID] = book
		}
	}

	var closeTime time.Time
	if m.EndDate != "" {
		if ts, err := time.Parse(time.RFC3339, m.EndDate); err == nil {
			closeTime = ts
		}
	}

	price := collectors.PriceSnapshot{
		YesBid: bestBid(orderbooks, clobIDs, 0),
		YesAsk: bestAsk(orderbooks, clobIDs, 0),
		NoBid:  bestBid(orderbooks, clobIDs, 1),
		NoAsk:  bestAsk(orderbooks, clobIDs, 1),
	}

	return collectors.Market{
		MarketID:     m.ID,
		Question:     m.Question,
		Subtitle:     m.Description,
		TickSize:     m.MinTickSize,
		CloseTime:    closeTime,
		Volume:       m.VolumeNum,
		Volume24h:    m.Volume24h,
		OpenInterest: m.OpenInterest,
		Price:        price,
		Orderbooks:   orderbooks,
		ClobTokenIDs: clobIDs,
	}
}

func bestBid(orderbooks map[string]collectors.Orderbook, clobIDs []string, idx int) float64 {
	if idx >= len(clobIDs) {
		return 0
	}
	if book, ok := orderbooks[clobIDs[idx]]; ok && len(book.Bids) > 0 {
		return book.Bids[0].Price
	}
	return 0
}

func bestAsk(orderbooks map[string]collectors.Orderbook, clobIDs []string, idx int) float64 {
	if idx >= len(clobIDs) {
		return 0
	}
	if book, ok := orderbooks[clobIDs[idx]]; ok && len(book.Asks) > 0 {
		return book.Asks[0].Price
	}
	return 0
}

func parseClobTokenIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return ids
}

func convertClobBook(b clobBook) collectors.Orderbook {
	out := collectors.Orderbook{}
	for _, lvl := range b.Bids {
		price := parseDecimal(lvl.Price)
		size := parseDecimal(lvl.Size)
		out.Bids = append(out.Bids, collectors.OrderbookLevel{
			Price:     price,
			Quantity:  size,
			RawPrice:  price,
			RawAmount: size,
		})
	}
	for _, lvl := range b.Asks {
		price := parseDecimal(lvl.Price)
		size := parseDecimal(lvl.Size)
		out.Asks = append(out.Asks, collectors.OrderbookLevel{
			Price:     price,
			Quantity:  size,
			RawPrice:  price,
			RawAmount: size,
		})
	}
	return out
}

func parseDecimal(val string) float64 {
	f, _ := strconv.ParseFloat(val, 64)
	return f
}

var placeholderQuestionRe = regexp.MustCompile(`(?i)^will\s+\w+\s+[a-z]\b`)

func isPlaceholderMarket(m *market) bool {
	q := strings.TrimSpace(m.Question)
	if placeholderQuestionRe.MatchString(q) {
		return true
	}
	desc := strings.ToLower(m.Description)
	if strings.Contains(desc, "may be updated to replace") || strings.Contains(desc, "placeholder") {
		return true
	}
	return false
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

func sleep(attempt int) {
	backoff := time.Duration(1<<uint(attempt-1)) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	time.Sleep(backoff)
}

type eventSummary struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Closed   bool   `json:"closed"`
	Category string `json:"category"`
}

type eventDetail struct {
	ID                    string   `json:"id"`
	Title                 string   `json:"title"`
	Description           string   `json:"description"`
	ResolutionSource      string   `json:"resolutionSource"`
	ResolutionDescription string   `json:"resolutionDescription"`
	Closed                bool     `json:"closed"`
	Category              string   `json:"category"`
	EndDate               string   `json:"endDate"`
	Markets               []market `json:"markets"`
}

type market struct {
	ID             string  `json:"id"`
	Question       string  `json:"question"`
	Description    string  `json:"description"`
	LastTradePrice float64 `json:"lastTradePrice"`
	VolumeNum      float64 `json:"volumeNum"`
	Volume24h      float64 `json:"volume24hr"`
	OpenInterest   float64 `json:"openInterest"`
	ClobTokenIds   string  `json:"clobTokenIds"`
	MinTickSize    float64 `json:"orderPriceMinTickSize"`
	EndDate        string  `json:"endDate"`
	Active         bool    `json:"active"`
	Closed         bool    `json:"closed"`
}

type clobBook struct {
	Bids []clobLevel `json:"bids"`
	Asks []clobLevel `json:"asks"`
}

type clobLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}
