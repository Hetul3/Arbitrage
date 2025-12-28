package workers

import (
	"strings"
	"time"

	"github.com/hetulpatel/Arbitrage/internal/models"
)

const maxSentences = 7

func buildEmbeddingText(snapshot *models.MarketSnapshot) string {
	var b strings.Builder

	if snapshot.Event.Title != "" {
		b.WriteString(snapshot.Event.Title)
	}

	if snapshot.Market.Question != "" && snapshot.Market.Question != snapshot.Event.Title {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(snapshot.Market.Question)
	} else if snapshot.Market.Question != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(snapshot.Market.Question)
	}

	closeTime := snapshot.Market.CloseTime
	if closeTime.IsZero() {
		closeTime = snapshot.Event.CloseTime
	}
	if !closeTime.IsZero() {
		b.WriteString("\nSettle date: ")
		b.WriteString(closeTime.Format(time.DateOnly))
	}

	if desc := trimSentences(snapshot.Event.Description, maxSentences/2); desc != "" {
		b.WriteString("\nDescription: ")
		b.WriteString(desc)
	}

	if sub := trimSentences(snapshot.Market.Subtitle, maxSentences); sub != "" {
		b.WriteString("\nSubtitle: ")
		b.WriteString(sub)
	}

	return strings.TrimSpace(b.String())
}

func trimSentences(text string, limit int) string {
	if limit <= 0 || strings.TrimSpace(text) == "" {
		return ""
	}
	sentences := splitSentences(text)
	if len(sentences) <= limit {
		return strings.Join(sentences, " ")
	}
	return strings.Join(sentences[:limit], " ")
}

func splitSentences(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	// naive sentence splitting
	delimiters := []string{". ", "? ", "! "}
	sentences := []string{text}
	for _, d := range delimiters {
		var tmp []string
		for _, segment := range sentences {
			parts := strings.Split(segment, d)
			for i, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				// add delimiter back except last part
				if i < len(parts)-1 {
					part += string([]rune(d)[0])
				}
				tmp = append(tmp, part)
			}
		}
		sentences = tmp
	}
	return sentences
}
