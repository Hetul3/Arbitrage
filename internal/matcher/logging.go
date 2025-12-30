package matcher

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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
	l.appendToFile(source, res.Target, res.Similarity, threshold)
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

func (l *Logger) appendToFile(source, target *models.MarketSnapshot, sim, threshold float64) {
	entry := map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"similarity": sim,
		"threshold":  threshold,
		"source":     source,
		"target":     target,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		fmt.Printf("[matcher] log file marshal error: %v\n", err)
		return
	}
	f, err := os.OpenFile("matches.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("[matcher] log file open error: %v\n", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		fmt.Printf("[matcher] log file write error: %v\n", err)
	}
}
