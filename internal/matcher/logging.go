package matcher

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hetulpatel/Arbitrage/internal/models"
)

type LogMode int

const (
	LogModeQuiet LogMode = iota
	LogModeSummary
	LogModeVerbose
)

func ParseLogMode(input string) LogMode {
	switch strings.ToLower(input) {
	case "summary":
		return LogModeSummary
	case "verbose":
		return LogModeVerbose
	default:
		return LogModeQuiet
	}
}

type Logger struct {
	mode LogMode
}

func NewLogger(mode LogMode) *Logger {
	return &Logger{mode: mode}
}

func (l *Logger) Mode() LogMode {
	if l == nil {
		return LogModeQuiet
	}
	return l.mode
}

func (l *Logger) Enabled() bool {
	return l != nil && l.mode != LogModeQuiet
}

func (l *Logger) LogMatch(source *models.MarketSnapshot, res *Result, threshold float64) {
	if !l.Enabled() || res == nil {
		return
	}
	switch l.mode {
	case LogModeSummary:
		fmt.Printf("[matcher] matched %s (%s) -> %s (%s) sim=%.4f threshold=%.4f\n",
			source.Venue, safeQuestion(source), res.Target.Venue, safeQuestion(res.Target), res.Similarity, threshold)
	case LogModeVerbose:
		srcJSON, _ := json.MarshalIndent(source, "", "  ")
		dstJSON, _ := json.MarshalIndent(res.Target, "", "  ")
		fmt.Printf("[matcher] match sim=%.4f threshold=%.4f\nsource=%s\nmatch=%s\n", res.Similarity, threshold, string(srcJSON), string(dstJSON))
	}
}

func safeQuestion(snap *models.MarketSnapshot) string {
	switch {
	case snap == nil:
		return ""
	case snap.Market.Question != "":
		return snap.Market.Question
	case snap.Event.Title != "":
		return snap.Event.Title
	default:
		return snap.Market.MarketID
	}
}
