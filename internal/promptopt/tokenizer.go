package promptopt

import (
	"strings"
	"unicode"
)

// Tokenizer provides token counting and optimization
type Tokenizer struct {
	// Simple word-based tokenizer (in production, use tiktoken or similar)
}

// NewTokenizer creates a new tokenizer
func NewTokenizer() *Tokenizer {
	return &Tokenizer{}
}

// CountTokens estimates token count (simplified approximation)
// In production, use tiktoken or model-specific tokenizer
func (t *Tokenizer) CountTokens(text string) int {
	// Rough approximation: 1 token ≈ 4 characters or 0.75 words
	words := strings.Fields(text)
	
	// Count words + punctuation as separate tokens
	tokenCount := 0
	for _, word := range words {
		// Each word is a token
		tokenCount++
		
		// Additional tokens for punctuation
		for _, char := range word {
			if !unicode.IsLetter(char) && !unicode.IsNumber(char) {
				tokenCount++
			}
		}
	}
	
	return tokenCount
}

// CompressText reduces text length while preserving meaning
func (t *Tokenizer) CompressText(text string, targetReduction float64) string {
	if targetReduction <= 0 || targetReduction >= 1 {
		return text
	}

	// Step 1: Remove filler words
	compressed := t.removeFillerWords(text)
	
	// Step 2: Replace verbose phrases
	compressed = t.replaceVerbosePhrases(compressed)
	
	// Step 3: Use shorter synonyms
	compressed = t.useShorterSynonyms(compressed)
	
	// Check if we've achieved target reduction
	originalTokens := t.CountTokens(text)
	currentTokens := t.CountTokens(compressed)
	reduction := 1.0 - float64(currentTokens)/float64(originalTokens)
	
	// If we haven't reduced enough, be more aggressive
	if reduction < targetReduction {
		compressed = t.aggressiveCompression(compressed)
	}
	
	return compressed
}

// removeFillerWords removes common filler words
func (t *Tokenizer) removeFillerWords(text string) string {
	fillerWords := []string{
		" really ", " very ", " quite ", " rather ", " somewhat ",
		" actually ", " basically ", " essentially ", " literally ",
		" simply ", " just ", " totally ", " completely ",
	}
	
	result := " " + text + " "
	for _, filler := range fillerWords {
		result = strings.ReplaceAll(result, filler, " ")
	}
	
	return strings.TrimSpace(result)
}

// replaceVerbosePhrases replaces verbose phrases with concise alternatives
func (t *Tokenizer) replaceVerbosePhrases(text string) string {
	replacements := map[string]string{
		"in order to":           "to",
		"due to the fact that":  "because",
		"at this point in time": "now",
		"for the purpose of":    "for",
		"in the event that":     "if",
		"with regard to":        "regarding",
		"it is important to":    "",
		"it should be noted":    "",
		"as a matter of fact":   "in fact",
		"take into consideration": "consider",
		"make a decision":       "decide",
		"come to a conclusion":  "conclude",
		"give consideration to": "consider",
		"is in agreement with":  "agrees with",
		"has the ability to":    "can",
		"prior to":              "before",
		"subsequent to":         "after",
	}
	
	result := text
	for verbose, concise := range replacements {
		result = strings.ReplaceAll(result, verbose, concise)
		// Also try capitalized versions
		capitalVerbose := strings.Title(verbose)
		capitalConcise := strings.Title(concise)
		result = strings.ReplaceAll(result, capitalVerbose, capitalConcise)
	}
	
	return result
}

// useShorterSynonyms replaces long words with shorter synonyms
func (t *Tokenizer) useShorterSynonyms(text string) string {
	synonyms := map[string]string{
		"accomplish":    "do",
		"additional":    "more",
		"assistance":    "help",
		"demonstrate":   "show",
		"implement":     "use",
		"facilitate":    "help",
		"utilize":       "use",
		"approximately": "about",
		"requirement":   "need",
		"sufficient":    "enough",
		"terminate":     "end",
		"endeavor":      "try",
		"magnitude":     "size",
		"numerous":      "many",
		"purchase":      "buy",
		"acquire":       "get",
	}
	
	words := strings.Fields(text)
	result := make([]string, len(words))
	
	for i, word := range words {
		lowerWord := strings.ToLower(word)
		if replacement, ok := synonyms[lowerWord]; ok {
			result[i] = replacement
		} else {
			result[i] = word
		}
	}
	
	return strings.Join(result, " ")
}

// aggressiveCompression applies more aggressive compression
func (t *Tokenizer) aggressiveCompression(text string) string {
	// Remove example phrases
	result := text
	
	// Remove "for example" phrases
	result = strings.ReplaceAll(result, "for example, ", "")
	result = strings.ReplaceAll(result, "for instance, ", "")
	result = strings.ReplaceAll(result, "such as ", "")
	
	// Remove redundant explanations in parentheses (keep only short ones)
	lines := strings.Split(result, "\n")
	compressed := []string{}
	
	for _, line := range lines {
		// Skip very short lines that might be redundant
		if len(strings.TrimSpace(line)) < 10 {
			continue
		}
		compressed = append(compressed, line)
	}
	
	return strings.Join(compressed, "\n")
}

// OptimizeToTokenBudget compresses text to fit within token budget
func (t *Tokenizer) OptimizeToTokenBudget(text string, maxTokens int) (string, int) {
	currentTokens := t.CountTokens(text)
	
	if currentTokens <= maxTokens {
		return text, currentTokens
	}
	
	// Calculate required reduction
	reduction := float64(currentTokens-maxTokens) / float64(currentTokens)
	
	// Apply compression
	compressed := t.CompressText(text, reduction)
	finalTokens := t.CountTokens(compressed)
	
	// If still over budget, truncate
	if finalTokens > maxTokens {
		words := strings.Fields(compressed)
		// Rough estimate: keep proportion of words
		keepWords := int(float64(len(words)) * float64(maxTokens) / float64(finalTokens))
		if keepWords > 0 && keepWords < len(words) {
			compressed = strings.Join(words[:keepWords], " ") + "..."
			finalTokens = t.CountTokens(compressed)
		}
	}
	
	return compressed, finalTokens
}

// CalculateTokenSavings calculates token savings between original and optimized
func (t *Tokenizer) CalculateTokenSavings(original, optimized string) (int, float64) {
	originalTokens := t.CountTokens(original)
	optimizedTokens := t.CountTokens(optimized)
	
	saved := originalTokens - optimizedTokens
	percentSaved := 0.0
	if originalTokens > 0 {
		percentSaved = float64(saved) / float64(originalTokens)
	}
	
	return saved, percentSaved
}

