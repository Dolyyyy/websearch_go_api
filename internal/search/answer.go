package search

import (
	"net/url"
	"sort"
	"strings"
	"unicode"
)

var stopWords = map[string]struct{}{
	"a":       {},
	"ai":      {},
	"au":      {},
	"aux":     {},
	"avec":    {},
	"ce":      {},
	"ces":     {},
	"dans":    {},
	"de":      {},
	"demain":  {},
	"des":     {},
	"du":      {},
	"en":      {},
	"est":     {},
	"et":      {},
	"jour":    {},
	"la":      {},
	"le":      {},
	"les":     {},
	"meteo":   {},
	"météo":   {},
	"ou":      {},
	"par":     {},
	"plus":    {},
	"pour":    {},
	"quel":    {},
	"quelle":  {},
	"sur":     {},
	"thouars": {},
	"un":      {},
	"une":     {},
	"weather": {},
	"what":    {},
}

func BuildAnswer(query string, results []Result) *Answer {
	if len(results) == 0 {
		return nil
	}

	type candidate struct {
		result Result
		score  int
	}

	queryTokens := tokens(query)
	candidates := make([]candidate, 0, len(results))
	for _, result := range results {
		candidates = append(candidates, candidate{
			result: result,
			score:  scoreResult(query, queryTokens, result),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	best := candidates[0].result
	text := buildAnswerText(best)
	if text == "" {
		return nil
	}

	return &Answer{
		Text:   text,
		Title:  best.Title,
		URL:    best.URL,
		Source: best.Provider,
	}
}

func scoreResult(query string, queryTokens []string, result Result) int {
	text := normalize(result.Title + " " + result.Description)
	score := 0

	for _, token := range queryTokens {
		if token == "" {
			continue
		}
		if strings.Contains(text, token) {
			score += 4
		}
		if strings.Contains(normalize(result.Title), token) {
			score += 6
		}
	}

	if looksLikeWeatherQuery(query) {
		score += weatherBonus(result)
	}

	if result.Description != "" {
		score += 3
	}

	return score
}

func weatherBonus(result Result) int {
	text := normalize(result.Title + " " + result.Description)
	bonus := 0

	for _, token := range []string{"demain", "heure", "temperature", "température", "pluie", "vent", "rafales", "meteo", "météo"} {
		if strings.Contains(text, normalize(token)) {
			bonus += 3
		}
	}

	if strings.Contains(result.Description, "°") {
		bonus += 8
	}

	host := hostOf(result.URL)
	for _, preferred := range []string{
		"meteofrance.com",
		"lachainemeteo.com",
		"meteocity.com",
		"meteoblue.com",
		"meteo.franceinfo.fr",
		"accuweather.com",
	} {
		if strings.Contains(host, preferred) {
			bonus += 5
			break
		}
	}

	return bonus
}

func buildAnswerText(result Result) string {
	description := strings.TrimSpace(result.Description)
	title := strings.TrimSpace(result.Title)

	switch {
	case description != "":
		return ensureSentence(description)
	case title != "":
		return ensureSentence(title)
	default:
		return ""
	}
}

func looksLikeWeatherQuery(query string) bool {
	norm := normalize(query)
	for _, token := range []string{"meteo", "météo", "weather", "temperature", "température", "pluie", "vent"} {
		if strings.Contains(norm, normalize(token)) {
			return true
		}
	}
	return false
}

func hostOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func tokens(s string) []string {
	fields := strings.FieldsFunc(normalize(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) < 2 {
			continue
		}
		if _, skip := stopWords[field]; skip {
			continue
		}
		out = append(out, field)
	}

	return out
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func ensureSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	last := s[len(s)-1]
	if last == '.' || last == '!' || last == '?' {
		return s
	}
	return s + "."
}
