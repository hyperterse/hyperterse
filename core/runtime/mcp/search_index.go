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
	// Conversational prompts often include filler words. Penalize unmatched query
	// tokens softly so strong keyword matches are still ranked highly.
	coverageMissPenalty = 0.10
)

type tokenSet map[string]struct{}

// scoreComponent is one scoring signal in the final relevance computation.
//
// score:
//   normalized similarity for this signal in [0..1]
// weight:
//   default importance before dynamic normalization
// active:
//   whether this signal should participate for the current tool
//
// Signals are marked inactive when the corresponding tool field is effectively
// unavailable. Example: handler-only tools have no meaningful SQL statement,
// so statement scoring is disabled entirely.
type scoreComponent struct {
	score  float64
	weight float64
	active bool
}

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

// newToolSearchIndex precomputes normalized text, token sets, and n-grams for
// each tool so runtime search only needs query-time scoring.
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
		// Handler-only tools (no adapter bindings) should not be ranked by SQL
		// statement content because statements are synthetic placeholders
		// (e.g., "SELECT 1") for executor compatibility.
		//
		// Clearing statementText here ensures statement tokens and statement bonus
		// are both disabled downstream.
		if len(tool.Use) == 0 {
			statementText = ""
		}
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

// scoreEntry computes the per-tool raw relevance in [0..1] using:
//   1) token-coverage signals across name/statement/inputs/description
//   2) character n-gram similarity over combined metadata
//   3) exact-substring bonus for full-query containment
//
// Important behavior:
// - Components without real data are marked inactive and excluded from both
//   numerator and denominator via weight normalization.
// - This avoids penalizing tool types that naturally omit certain fields
//   (e.g., handler-only tools with no adapter statement or inputs).
func scoreEntry(query string, queryTokens []string, queryNGrams tokenSet, entry searchIndexEntry) float64 {
	nameScore := coverageScore(queryTokens, entry.nameTokens)
	descriptionScore := coverageScore(queryTokens, entry.descriptionTokens)
	statementScore := coverageScore(queryTokens, entry.statementTokens)
	inputNameScore := coverageScore(queryTokens, entry.inputNameTokens)
	inputDescriptionScore := coverageScore(queryTokens, entry.inputDescriptionTokens)
	ngramScore := jaccardSimilarity(queryNGrams, entry.combinedNGrams)

	raw := combineComponentScores([]scoreComponent{
		{score: nameScore, weight: 0.30, active: len(entry.nameTokens) > 0},
		{score: statementScore, weight: 0.25, active: len(entry.statementTokens) > 0},
		{score: inputNameScore, weight: 0.18, active: len(entry.inputNameTokens) > 0},
		{score: descriptionScore, weight: 0.14, active: len(entry.descriptionTokens) > 0},
		{score: inputDescriptionScore, weight: 0.08, active: len(entry.inputDescriptionTokens) > 0},
		{score: ngramScore, weight: 0.05, active: len(entry.combinedNGrams) > 0},
	})

	// Bonus stage prefers explicit full-query phrase containment, with priority:
	// name > statement > description.
	// Statement bonus is guarded by statementTokens so handler-only placeholders
	// never accidentally receive statement credit.
	if strings.Contains(entry.nameText, query) {
		raw += 0.10
	} else if len(entry.statementTokens) > 0 && strings.Contains(entry.statementText, query) {
		raw += 0.08
	} else if strings.Contains(entry.descriptionText, query) {
		raw += 0.03
	}

	if raw > 1 {
		return 1
	}
	return raw
}

// combineComponentScores normalizes active component weights to sum to 1.0,
// then returns the weighted sum of component scores.
//
// This makes weighting consistent across heterogeneous tools:
// - if statement/input fields are absent, their weight is redistributed to the
//   remaining active signals instead of reducing the final score ceiling.
func combineComponentScores(components []scoreComponent) float64 {
	normalizedComponents := normalizeActiveComponentWeights(components)
	weightedScore := 0.0
	for _, component := range normalizedComponents {
		if !component.active {
			continue
		}
		weightedScore += component.score * component.weight
	}

	return weightedScore
}

// normalizeActiveComponentWeights rescales only active component weights so
// active weights sum to exactly 1.0.
//
// Inactive components are always forced to weight 0.
// If no component is active, all weights remain 0 and the caller receives 0.
func normalizeActiveComponentWeights(components []scoreComponent) []scoreComponent {
	out := make([]scoreComponent, len(components))
	copy(out, components)

	totalWeight := 0.0
	for _, component := range out {
		if !component.active {
			continue
		}
		totalWeight += component.weight
	}

	if totalWeight <= 0 {
		for i := range out {
			out[i].weight = 0
		}
		return out
	}

	for i := range out {
		if !out[i].active {
			out[i].weight = 0
			continue
		}
		out[i].weight = out[i].weight / totalWeight
	}

	return out
}

// toRelevanceScore maps raw [0..1] to user-facing integer [1..100].
//
// We intentionally reserve 0 (never returned) to simplify clients that treat
// 0 as "unset" and to keep every returned hit explicitly non-zero relevance.
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

// coverageScore measures token overlap quality between query and target.
//
// Match credit:
// - exact token match:   +1.0
// - partial token match: +0.6 (prefix/contains both directions)
// - miss: no direct score, but contributes soft denominator penalty
//
// Final form:
//   score / (score + misses * coverageMissPenalty)
//
// This "soft miss penalty" handles conversational queries better than dividing
// by query token count. Strong intent tokens (e.g., "weather") remain dominant
// even when prompts include filler language.
func coverageScore(queryTokens []string, target tokenSet) float64 {
	if len(queryTokens) == 0 || len(target) == 0 {
		return 0
	}

	score := 0.0
	misses := 0.0
	for _, queryToken := range queryTokens {
		if _, exact := target[queryToken]; exact {
			score += 1
			continue
		}
		if hasPartialTokenMatch(queryToken, target) {
			score += 0.6
			continue
		}
		misses += 1
	}

	if score <= 0 {
		return 0
	}

	denominator := score + (misses * coverageMissPenalty)
	if denominator <= 0 {
		return 0
	}

	return score / denominator
}

// hasPartialTokenMatch allows fuzzy lexical overlap without a full edit-distance
// engine. It is intentionally simple and fast for small metadata vocabularies.
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

// buildNGramSet creates character-level n-grams over whitespace-collapsed text.
// N-grams provide resilience when tokenization alone misses similarity.
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

// jaccardSimilarity computes intersection/union over n-gram sets.
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
