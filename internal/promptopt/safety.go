package promptopt

import (
	"regexp"
	"strings"
)

// SafetyChecker checks prompts for unsafe content
type SafetyChecker struct {
	unsafePatterns []*regexp.Regexp
	biasedTerms    []string
}

// NewSafetyChecker creates a new safety checker
func NewSafetyChecker() *SafetyChecker {
	sc := &SafetyChecker{
		unsafePatterns: make([]*regexp.Regexp, 0),
		biasedTerms: []string{
			// Common biased terms (simplified list)
			"illegal", "hack", "exploit", "weaponize",
		},
	}
	
	sc.initializePatterns()
	return sc
}

// initializePatterns initializes unsafe content patterns
func (sc *SafetyChecker) initializePatterns() {
	unsafeStrings := []string{
		`(?i)(how to (build|create|make).*(bomb|weapon|explosive))`,
		`(?i)(bypass|circumvent|disable).*(security|authentication|firewall)`,
		`(?i)(steal|extract|obtain).*(password|credential|private key)`,
		`(?i)(generate|create).*(malware|virus|trojan)`,
	}
	
	for _, pattern := range unsafeStrings {
		if re, err := regexp.Compile(pattern); err == nil {
			sc.unsafePatterns = append(sc.unsafePatterns, re)
		}
	}
}

// CheckSafety checks if a prompt is safe
func (sc *SafetyChecker) CheckSafety(prompt string) (bool, string) {
	// Check for unsafe patterns
	for _, pattern := range sc.unsafePatterns {
		if pattern.MatchString(prompt) {
			return true, "Contains potentially harmful content"
		}
	}
	
	// Check for injection attempts
	if sc.detectInjection(prompt) {
		return true, "Potential prompt injection detected"
	}
	
	// Check for excessive bias
	if sc.detectBias(prompt) {
		return true, "Contains biased language"
	}
	
	return false, ""
}

// detectInjection checks for prompt injection attempts
func (sc *SafetyChecker) detectInjection(prompt string) bool {
	injectionPatterns := []string{
		"ignore previous instructions",
		"disregard all",
		"forget everything",
		"new instructions:",
		"system: ",
		"admin mode",
	}
	
	lowerPrompt := strings.ToLower(prompt)
	for _, pattern := range injectionPatterns {
		if strings.Contains(lowerPrompt, pattern) {
			return true
		}
	}
	
	return false
}

// detectBias checks for potentially biased language
func (sc *SafetyChecker) detectBias(prompt string) bool {
	// Count biased terms
	biasCount := 0
	lowerPrompt := strings.ToLower(prompt)
	
	for _, term := range sc.biasedTerms {
		if strings.Contains(lowerPrompt, term) {
			biasCount++
		}
	}
	
	// If more than 2 biased terms, flag as potentially biased
	return biasCount > 2
}

// RemoveBias attempts to remove biased language
func (sc *SafetyChecker) RemoveBias(prompt string) string {
	// Simple bias mitigation (in production, would use LLM-based rewriting)
	biasReplacements := map[string]string{
		"guys":        "everyone",
		"mankind":     "humanity",
		"manpower":    "workforce",
		"policeman":   "police officer",
		"fireman":     "firefighter",
	}
	
	result := prompt
	for biased, neutral := range biasReplacements {
		result = strings.ReplaceAll(result, biased, neutral)
	}
	
	return result
}

