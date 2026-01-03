package matches

import (
	"fmt"
	"sort"

	"github.com/hetulpatel/Arbitrage/internal/hashutil"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

func textDigest(snap *models.MarketSnapshot) string {
	if snap == nil {
		return ""
	}
	return hashutil.HashStrings(
		snap.Event.Title,
		snap.Market.Question,
		snap.Market.Subtitle,
		snap.Event.Description,
		snap.Event.ResolutionSource,
		snap.Event.ResolutionDetails,
		snap.Event.ContractTermsURL,
	)
}

// VerdictCacheKey builds an order-independent cache key for a pair based on venue, market, and text hash.
func VerdictCacheKey(a, b *models.MarketSnapshot) string {
	if a == nil || b == nil {
		return ""
	}
	left := fmt.Sprintf("%s:%s:%s", a.Venue, a.Market.MarketID, textDigest(a))
	right := fmt.Sprintf("%s:%s:%s", b.Venue, b.Market.MarketID, textDigest(b))
	parts := []string{left, right}
	sort.Strings(parts)
	return fmt.Sprintf("%s|%s", parts[0], parts[1])
}
