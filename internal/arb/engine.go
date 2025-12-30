package arb

import (
	"math"
	"sort"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/matches"
	"github.com/hetulpatel/Arbitrage/internal/models"
)

type Config struct {
	BudgetUSD float64
}

type Result struct {
	Opportunities map[matches.Direction]*matches.Opportunity
	Best          *matches.Opportunity
}

const epsilon = 1e-9

// Evaluate computes both arbitrage directions for a matched pair.
func Evaluate(match *matches.Payload, cfg Config) Result {
	if cfg.BudgetUSD <= 0 {
		cfg.BudgetUSD = 100
	}
	res := Result{Opportunities: make(map[matches.Direction]*matches.Opportunity)}

	pmSnap, kxSnap := extractSnapshots(match)
	if pmSnap == nil || kxSnap == nil {
		return res
	}

	opA := simulateDirection(cfg.BudgetUSD, matches.DirectionBuyYesPMBuyNoKalshi, pmSnap, kxSnap)
	if opA != nil {
		res.Opportunities[opA.Direction] = opA
		if res.Best == nil || opA.ProfitUSD > res.Best.ProfitUSD {
			res.Best = opA
		}
	}

	opB := simulateDirection(cfg.BudgetUSD, matches.DirectionBuyNoPMBuyYesKalshi, pmSnap, kxSnap)
	if opB != nil {
		res.Opportunities[opB.Direction] = opB
		if res.Best == nil || opB.ProfitUSD > res.Best.ProfitUSD {
			res.Best = opB
		}
	}

	return res
}

func extractSnapshots(match *matches.Payload) (*models.MarketSnapshot, *models.MarketSnapshot) {
	if match == nil {
		return nil, nil
	}
	var pm, kx *models.MarketSnapshot
	if match.Source.Venue == collectors.VenuePolymarket {
		pm = &match.Source
	} else if match.Source.Venue == collectors.VenueKalshi {
		kx = &match.Source
	}
	if match.Target.Venue == collectors.VenuePolymarket {
		pm = &match.Target
	} else if match.Target.Venue == collectors.VenueKalshi {
		kx = &match.Target
	}
	if pm == nil || kx == nil {
		return nil, nil
	}
	if pm.Venue != collectors.VenuePolymarket || kx.Venue != collectors.VenueKalshi {
		return nil, nil
	}
	return pm, kx
}

func simulateDirection(budget float64, dir matches.Direction, pmSnap, kxSnap *models.MarketSnapshot) *matches.Opportunity {
	if pmSnap == nil || kxSnap == nil {
		return nil
	}
	pmMarket := pmSnap.Market
	kxMarket := kxSnap.Market

	var pmBook, kxBook collectors.Orderbook
	var pmOutcome, kxOutcome string

	switch dir {
	case matches.DirectionBuyYesPMBuyNoKalshi:
		pmBook = getPMOrderbook(&pmMarket, true)
		kxBook = kxMarket.Orderbooks["no"]
		pmOutcome = "yes"
		kxOutcome = "no"
	case matches.DirectionBuyNoPMBuyYesKalshi:
		pmBook = getPMOrderbook(&pmMarket, false)
		kxBook = kxMarket.Orderbooks["yes"]
		pmOutcome = "no"
		kxOutcome = "yes"
	default:
		return nil
	}

	if len(pmBook.Asks) == 0 || len(kxBook.Asks) == 0 {
		return nil
	}

	pmIter := newAskIterator(pmBook.Asks)
	kxIter := newAskIterator(kxBook.Asks)

	totalQty := 0.0
	polyCost := 0.0
	kalshiCost := 0.0
	kalshiFees := 0.0

	for {
		yesQty := pmIter.peekQty()
		noQty := kxIter.peekQty()
		if yesQty <= epsilon || noQty <= epsilon {
			break
		}
		pricePM := pmIter.peekPrice()
		priceKX := kxIter.peekPrice()
		budgetRemaining := budget - (polyCost + kalshiCost + kalshiFees)
		if budgetRemaining <= epsilon {
			break
		}
		estimatedCost := pricePM + priceKX
		if estimatedCost <= epsilon {
			break
		}
		delta := math.Min(yesQty, noQty)
		delta = math.Min(delta, budgetRemaining/estimatedCost)
		if delta <= epsilon {
			break
		}

		costPM, ok := pmIter.take(delta)
		if !ok {
			break
		}
		costKX, ok := kxIter.take(delta)
		if !ok {
			break
		}

		fee := calcKalshiTakerFee(delta, priceKX)
		polyCost += costPM
		kalshiCost += costKX
		kalshiFees += fee
		totalQty += delta
		if budget-(polyCost+kalshiCost+kalshiFees) <= epsilon {
			break
		}
	}

	if totalQty <= epsilon {
		return nil
	}

	op := &matches.Opportunity{
		Direction:         dir,
		Quantity:          totalQty,
		PolymarketFeesUSD: 0,
		KalshiFeesUSD:     kalshiFees,
		BudgetUSD:         budget,
	}
	op.TotalCostUSD = polyCost + kalshiCost + kalshiFees
	op.ProfitUSD = totalQty - op.TotalCostUSD

	pmLeg := matches.Leg{
		Venue:    "polymarket",
		Side:     "buy",
		Outcome:  pmOutcome,
		Quantity: totalQty,
		CostUSD:  polyCost,
	}
	if totalQty > 0 {
		pmLeg.AvgPrice = polyCost / totalQty
	}
	kxLeg := matches.Leg{
		Venue:    "kalshi",
		Side:     "buy",
		Outcome:  kxOutcome,
		Quantity: totalQty,
		CostUSD:  kalshiCost,
	}
	if totalQty > 0 {
		kxLeg.AvgPrice = kalshiCost / totalQty
	}
	op.Legs = []matches.Leg{pmLeg, kxLeg}
	return op
}

func getPMOrderbook(m *collectors.Market, yes bool) collectors.Orderbook {
	if m == nil {
		return collectors.Orderbook{}
	}
	idx := 0
	if !yes {
		idx = 1
	}
	if len(m.ClobTokenIDs) <= idx {
		return collectors.Orderbook{}
	}
	tokenID := m.ClobTokenIDs[idx]
	if tokenID == "" {
		return collectors.Orderbook{}
	}
	if ob, ok := m.Orderbooks[tokenID]; ok {
		return ob
	}
	return collectors.Orderbook{}
}

type askIterator struct {
	levels []collectors.OrderbookLevel
	idx    int
}

func newAskIterator(levels []collectors.OrderbookLevel) *askIterator {
	copied := make([]collectors.OrderbookLevel, len(levels))
	copy(copied, levels)
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].Price < copied[j].Price
	})
	return &askIterator{levels: copied}
}

func (it *askIterator) peekQty() float64 {
	for it.idx < len(it.levels) {
		if qty := it.levels[it.idx].Quantity; qty > epsilon {
			return qty
		}
		it.idx++
	}
	return 0
}

func (it *askIterator) peekPrice() float64 {
	for it.idx < len(it.levels) {
		if qty := it.levels[it.idx].Quantity; qty > epsilon {
			return it.levels[it.idx].Price
		}
		it.idx++
	}
	return 0
}

func (it *askIterator) take(q float64) (float64, bool) {
	for it.idx < len(it.levels) {
		lvl := &it.levels[it.idx]
		if lvl.Quantity <= epsilon {
			it.idx++
			continue
		}
		if lvl.Quantity+epsilon < q {
			return 0, false
		}
		lvl.Quantity -= q
		cost := q * lvl.Price
		if lvl.Quantity <= epsilon {
			it.idx++
		}
		return cost, true
	}
	return 0, false
}

func calcKalshiTakerFee(quantity, price float64) float64 {
	raw := 0.07 * quantity * price * (1 - price)
	return math.Ceil(raw*100) / 100
}
