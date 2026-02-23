package mcp

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
)

const (
	defaultSearchLimit = 10
	searchNGramSize    = 3
)

type tokenSet map[string]struct{}

type searchHit struct {
	Tool           *hyperterse.Tool
	RelevanceScore int
	rawScore       float64
}

type searchIndexEntry struct {
	tool                   *hyperterse.Tool
	nameText               string
	descriptionText        string
	statementText          string
	nameTokens             tokenSet
	descriptionTokens      tokenSet
	statementTokens        tokenSet
	inputNameTokens        tokenSet
	inputDescriptionTokens tokenSet
	combinedNGrams         tokenSet
}

type toolSearchIndex struct {
	entries []searchIndexEntry
}

func newToolSearchIndex(tools []*hyperterse.Tool) *toolSearchIndex {
	index := &toolSearchIndex{
		entries: make([]searchIndexEntry, 0, len(tools)),
	}
	for _, tool := range tools {
		if tool == nil {
			continue
		}

		nameText := normalizeText(tool.Name)
		descriptionText := normalizeText(tool.Description)
		statementText := normalizeText(tool.Statement)
		inputNamesText, inputDescriptionsText := normalizeInputText(tool.Inputs)
		combinedText := strings.TrimSpace(strings.Join(
			[]string{nameText, descriptionText, statementText, inputNamesText, inputDescriptionsText},
			" ",
		))

		index.entries = append(index.entries, searchIndexEntry{
			tool:                   tool,
			nameText:               nameText,
			descriptionText:        descriptionText,
			statementText:          statementText,
			nameTokens:             toTokenSet(tokenize(nameText)),
			descriptionTokens:      toTokenSet(tokenize(descriptionText)),
			statementTokens:        toTokenSet(tokenize(statementText)),
			inputNameTokens:        toTokenSet(tokenize(inputNamesText)),
			inputDescriptionTokens: toTokenSet(tokenize(inputDescriptionsText)),
			combinedNGrams:         buildNGramSet(combinedText, searchNGramSize),
		})
	}

	return index
}

func (idx *toolSearchIndex) Search(query string, limit int) []searchHit {
	if idx == nil {
		return nil
	}

	normalizedQuery := normalizeText(query)
	if normalizedQuery == "" {
		return nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	queryTokens := tokenize(normalizedQuery)
	queryNGrams := buildNGramSet(normalizedQuery, searchNGramSize)

	hits := make([]searchHit, 0, len(idx.entries))
	for _, entry := range idx.entries {
		rawScore := scoreEntry(normalizedQuery, queryTokens, queryNGrams, entry)
		if rawScore <= 0 {
			continue
		}

		hits = append(hits, searchHit{
			Tool:           entry.tool,
			RelevanceScore: toRelevanceScore(rawScore),
			rawScore:       rawScore,
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].rawScore == hits[j].rawScore {
			if hits[i].Tool.Name == hits[j].Tool.Name {
				return hits[i].RelevanceScore > hits[j].RelevanceScore
			}
			return hits[i].Tool.Name < hits[j].Tool.Name
		}
		return hits[i].rawScore > hits[j].rawScore
	})

	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func scoreEntry(query string, queryTokens []string, queryNGrams tokenSet, entry searchIndexEntry) float64 {
	nameScore := coverageScore(queryTokens, entry.nameTokens)
	descriptionScore := coverageScore(queryTokens, entry.descriptionTokens)
	statementScore := coverageScore(queryTokens, entry.statementTokens)
	inputNameScore := coverageScore(queryTokens, entry.inputNameTokens)
	inputDescriptionScore := coverageScore(queryTokens, entry.inputDescriptionTokens)
	ngramScore := jaccardSimilarity(queryNGrams, entry.combinedNGrams)

	raw := (nameScore * 0.30) +
		(statementScore * 0.25) +
		(inputNameScore * 0.18) +
		(descriptionScore * 0.14) +
		(inputDescriptionScore * 0.08) +
		(ngramScore * 0.05)

	if strings.Contains(entry.nameText, query) {
		raw += 0.10
	} else if strings.Contains(entry.statementText, query) {
		raw += 0.08
	} else if strings.Contains(entry.descriptionText, query) {
		raw += 0.03
	}

	if raw > 1 {
		return 1
	}
	return raw
}

func toRelevanceScore(rawScore float64) int {
	scaled := int(math.Round(rawScore*99)) + 1
	if scaled < 1 {
		return 1
	}
	if scaled > 100 {
		return 100
	}
	return scaled
}

func normalizeInputText(inputs []*hyperterse.Input) (string, string) {
	nameParts := make([]string, 0, len(inputs))
	descriptionParts := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}

		normalizedName := normalizeText(input.Name)
		if normalizedName != "" {
			nameParts = append(nameParts, normalizedName)
		}

		normalizedDescription := normalizeText(input.Description)
		if normalizedDescription != "" {
			descriptionParts = append(descriptionParts, normalizedDescription)
		}
	}

	return strings.Join(nameParts, " "), strings.Join(descriptionParts, " ")
}

func normalizeText(value string) string {
	value = strings.ToLower(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(value))
	previousWasSpace := true

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			previousWasSpace = false
			continue
		}
		if !previousWasSpace {
			b.WriteByte(' ')
			previousWasSpace = true
		}
	}

	return strings.TrimSpace(b.String())
}

func tokenize(value string) []string {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return nil
	}

	seen := make(tokenSet, len(parts))
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		tokens = append(tokens, part)
	}
	return tokens
}

func toTokenSet(tokens []string) tokenSet {
	if len(tokens) == 0 {
		return nil
	}
	out := make(tokenSet, len(tokens))
	for _, token := range tokens {
		out[token] = struct{}{}
	}
	return out
}

func coverageScore(queryTokens []string, target tokenSet) float64 {
	if len(queryTokens) == 0 || len(target) == 0 {
		return 0
	}

	score := 0.0
	for _, queryToken := range queryTokens {
		if _, exact := target[queryToken]; exact {
			score += 1
			continue
		}
		if hasPartialTokenMatch(queryToken, target) {
			score += 0.6
		}
	}

	return score / float64(len(queryTokens))
}

func hasPartialTokenMatch(queryToken string, target tokenSet) bool {
	for token := range target {
		if strings.HasPrefix(token, queryToken) ||
			strings.HasPrefix(queryToken, token) ||
			strings.Contains(token, queryToken) ||
			strings.Contains(queryToken, token) {
			return true
		}
	}
	return false
}

func buildNGramSet(value string, n int) tokenSet {
	if value == "" {
		return nil
	}

	collapsed := strings.ReplaceAll(value, " ", "")
	runes := []rune(collapsed)
	if len(runes) == 0 {
		return nil
	}
	if len(runes) <= n {
		return tokenSet{string(runes): {}}
	}

	out := make(tokenSet, len(runes)-n+1)
	for i := 0; i <= len(runes)-n; i++ {
		out[string(runes[i:i+n])] = struct{}{}
	}
	return out
}

func jaccardSimilarity(a, b tokenSet) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	intersection := 0
	for token := range a {
		if _, ok := b[token]; ok {
			intersection++
		}
	}

	if intersection == 0 {
		return 0
	}

	union := len(a) + len(b) - intersection
	return float64(intersection) / float64(union)
}
