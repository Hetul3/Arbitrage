package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/matches"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

type promptPayload struct {
	PairID         string        `json:"pair_id"`
	MatchedAtUTC   string        `json:"matched_at_utc"`
	GeneratedAtUTC string        `json:"generated_at_utc"`
	Similarity     float64       `json:"similarity"`
	Distance       float64       `json:"distance"`
	Polymarket     marketPayload `json:"polymarket"`
	Kalshi         marketPayload `json:"kalshi"`
	Notes          []string      `json:"notes,omitempty"`
}

type marketPayload struct {
	Venue             string                        `json:"venue"`
	EventID           string                        `json:"event_id"`
	MarketID          string                        `json:"market_id"`
	Title             string                        `json:"event_title,omitempty"`
	Question          string                        `json:"question"`
	Subtitle          string                        `json:"subtitle,omitempty"`
	Description       string                        `json:"description,omitempty"`
	ResolutionSource  string                        `json:"resolution_source,omitempty"`
	ResolutionDetails string                        `json:"resolution_details,omitempty"`
	CloseTimeUTC      string                        `json:"close_time_utc,omitempty"`
	Category          string                        `json:"category,omitempty"`
	SettlementSources []collectors.ResolutionSource `json:"settlement_sources,omitempty"`
	ReferenceURL      string                        `json:"reference_url,omitempty"`
	ContractTermsURL  string                        `json:"contract_terms_url,omitempty"`
	ContractText      string                        `json:"contract_terms_excerpt,omitempty"`
	OutcomeMapping    outcomeMapping                `json:"outcome_mapping"`
	DataSourceDomains []string                      `json:"data_source_domains,omitempty"`
}

type outcomeMapping struct {
	Yes string `json:"yes_means"`
	No  string `json:"no_means"`
}

func buildPromptPayload(ctx context.Context, payload *matches.Payload, pdfExtractor PDFExtractor) (*promptPayload, error) {
	pmSnap, kxSnap := orderSnapshots(payload)
	if pmSnap == nil || kxSnap == nil {
		return nil, fmt.Errorf("validator: payload missing polymarket or kalshi snapshot")
	}

	pmMarket := buildMarketPayload(pmSnap, pdfSection{})
	kxSection := pdfSection{}
	if pdfExtractor != nil && kxSnap.Event.ContractTermsURL != "" {
		if text, err := pdfExtractor.Extract(ctx, kxSnap.Event.ContractTermsURL); err == nil {
			kxSection.Text = truncateText(text, 6000)
		}
	}
	kxMarket := buildMarketPayload(kxSnap, kxSection)

	return &promptPayload{
		PairID:         payload.PairID,
		MatchedAtUTC:   formatTime(payload.MatchedAt),
		GeneratedAtUTC: formatTime(time.Now().UTC()),
		Similarity:     payload.Similarity,
		Distance:       payload.Distance,
		Polymarket:     pmMarket,
		Kalshi:         kxMarket,
	}, nil
}

type pdfSection struct {
	Text string
}

func buildMarketPayload(snap *models.MarketSnapshot, pdf pdfSection) marketPayload {
	event := snap.Event
	market := snap.Market
	closeTime := formatTime(market.CloseTime)
	if closeTime == "" {
		closeTime = formatTime(event.CloseTime)
	}

	settlement := event.SettlementSources
	if len(settlement) == 0 && event.ResolutionSource != "" {
		settlement = append(settlement, collectors.ResolutionSource{Name: event.ResolutionSource})
	}

	outcome := outcomeMapping{
		Yes: buildOutcomeText(market, true),
		No:  buildOutcomeText(market, false),
	}

	dataSourceDomains := collectDomains(settlement, market.ReferenceURL, event.ContractTermsURL, event.Description, event.ResolutionDetails, market.Subtitle)

	contractText := pdf.Text
	if contractText == "" {
		contractText = event.ResolutionDetails
	}

	return marketPayload{
		Venue:             string(snap.Venue),
		EventID:           event.EventID,
		MarketID:          market.MarketID,
		Title:             event.Title,
		Question:          market.Question,
		Subtitle:          market.Subtitle,
		Description:       event.Description,
		ResolutionSource:  event.ResolutionSource,
		ResolutionDetails: event.ResolutionDetails,
		CloseTimeUTC:      closeTime,
		Category:          event.Category,
		SettlementSources: settlement,
		ReferenceURL:      market.ReferenceURL,
		ContractTermsURL:  event.ContractTermsURL,
		ContractText:      contractText,
		OutcomeMapping:    outcome,
		DataSourceDomains: dataSourceDomains,
	}
}

func buildOutcomeText(m collectors.Market, yes bool) string {
	base := strings.TrimSpace(m.Question)
	if yes {
		if m.Subtitle != "" {
			return fmt.Sprintf("YES when: %s", strings.TrimSpace(m.Subtitle))
		}
		return fmt.Sprintf("YES when the question \"%s\" resolves positively.", base)
	}
	return "NO covers all other outcomes or when the YES condition fails."
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func orderSnapshots(payload *matches.Payload) (*models.MarketSnapshot, *models.MarketSnapshot) {
	if payload == nil {
		return nil, nil
	}
	var pm, kx *models.MarketSnapshot
	if payload.Source.Venue == collectors.VenuePolymarket {
		pm = &payload.Source
	} else if payload.Source.Venue == collectors.VenueKalshi {
		kx = &payload.Source
	}
	if payload.Target.Venue == collectors.VenuePolymarket {
		pm = &payload.Target
	} else if payload.Target.Venue == collectors.VenueKalshi {
		kx = &payload.Target
	}
	return pm, kx
}

func collectDomains(sources []collectors.ResolutionSource, urls ...string) []string {
	domainSet := make(map[string]struct{})
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return
		}
		host := strings.ToLower(u.Host)
		domainSet[host] = struct{}{}
	}
	for _, s := range sources {
		add(s.URL)
	}
	for _, raw := range urls {
		if raw == "" {
			continue
		}
		add(raw)
		for _, match := range urlRegex.FindAllString(raw, -1) {
			add(match)
		}
	}
	domains := make([]string, 0, len(domainSet))
	for host := range domainSet {
		domains = append(domains, host)
	}
	sort.Strings(domains)
	return domains
}

func truncateText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + " ... (truncated)"
}

func parseResult(raw string) (*Result, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("validator: empty llm response")
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}
	var res Result
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, err
	}
	return &res, nil
}
