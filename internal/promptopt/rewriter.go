package promptopt

import (
	"fmt"
	"strings"
)

// Rewriter applies prompt rewriting techniques
type Rewriter struct {
	techniques map[string]RewriteFunc
}

// RewriteFunc is a function that rewrites a prompt
type RewriteFunc func(prompt string, req *OptimizationRequest) (string, []string)

// NewRewriter creates a new rewriter with all techniques
func NewRewriter() *Rewriter {
	r := &Rewriter{
		techniques: make(map[string]RewriteFunc),
	}
	
	r.registerTechniques()
	return r
}

// registerTechniques registers all available rewriting techniques
func (r *Rewriter) registerTechniques() {
	r.techniques["chain-of-thought"] = r.applyChainOfThought
	r.techniques["few-shot"] = r.applyFewShot
	r.techniques["context-enrichment"] = r.applyContextEnrichment
	r.techniques["clarity"] = r.applyClarityEnhancement
	r.techniques["specificity"] = r.applySpecificity
}

// Rewrite applies appropriate rewriting techniques
func (r *Rewriter) Rewrite(req *OptimizationRequest) (string, []string, []string, error) {
	if req.OriginalPrompt == "" {
		return "", nil, nil, fmt.Errorf("empty prompt")
	}

	optimized := req.OriginalPrompt
	appliedTechniques := []string{}
	changes := []string{}
	var changesMade []string

	// Determine which techniques to apply
	techniquesToApply := r.selectTechniques(req)

	// Apply each technique
	for _, techniqueName := range techniquesToApply {
		if fn, ok := r.techniques[techniqueName]; ok {
			result, changesFromTechnique := fn(optimized, req)
			if result != optimized {
				optimized = result
				appliedTechniques = append(appliedTechniques, techniqueName)
				changes = append(changes, changesFromTechnique...)
			}
		}
	}

	// If no changes were made, at least apply clarity enhancement
	if len(appliedTechniques) == 0 {
		optimized, changesMade = r.applyClarityEnhancement(optimized, req)
		appliedTechniques = append(appliedTechniques, "clarity")
		changes = append(changes, changesMade...)
	}

	return optimized, appliedTechniques, changes, nil
}

// selectTechniques determines which techniques to apply based on request
func (r *Rewriter) selectTechniques(req *OptimizationRequest) []string {
	// If specific techniques requested, use those
	if len(req.Techniques) > 0 {
		return req.Techniques
	}

	techniques := []string{}

	// Task-specific technique selection
	switch req.TaskType {
	case "reasoning", "math", "complex":
		techniques = append(techniques, "chain-of-thought", "specificity")
	case "classification", "labeling":
		techniques = append(techniques, "few-shot", "clarity")
	case "factual", "qa":
		techniques = append(techniques, "context-enrichment", "specificity")
	default:
		// Apply general-purpose techniques
		techniques = append(techniques, "clarity", "specificity")
	}

	return techniques
}

// applyChainOfThought adds step-by-step reasoning structure
func (r *Rewriter) applyChainOfThought(prompt string, req *OptimizationRequest) (string, []string) {
	changes := []string{}
	
	// Check if already has step-by-step structure
	if strings.Contains(strings.ToLower(prompt), "step by step") ||
	   strings.Contains(strings.ToLower(prompt), "first,") ||
	   strings.Contains(prompt, "1.") {
		return prompt, changes
	}

	// Add chain-of-thought instruction
	enhanced := prompt + "\n\nLet's approach this step by step:\n" +
		"1. First, identify the key elements\n" +
		"2. Then, analyze each element\n" +
		"3. Finally, synthesize the findings into a comprehensive answer"
	
	changes = append(changes, "Added chain-of-thought reasoning structure")
	return enhanced, changes
}

// applyFewShot adds example-based learning (simplified - would query episodic memory in full implementation)
func (r *Rewriter) applyFewShot(prompt string, req *OptimizationRequest) (string, []string) {
	changes := []string{}
	
	// Check if already has examples
	if strings.Contains(strings.ToLower(prompt), "example") ||
	   strings.Contains(strings.ToLower(prompt), "for instance") {
		return prompt, changes
	}

	// Add few-shot instruction
	enhanced := "Here are examples of the expected output format:\n\n" +
		"Example 1: [Sample input] → [Sample output]\n" +
		"Example 2: [Sample input] → [Sample output]\n\n" +
		"Now for your task:\n" + prompt
	
	changes = append(changes, "Added few-shot learning examples")
	return enhanced, changes
}

// applyContextEnrichment adds relevant context
func (r *Rewriter) applyContextEnrichment(prompt string, req *OptimizationRequest) (string, []string) {
	changes := []string{}
	
	if req.Context == "" {
		return prompt, changes
	}

	enhanced := "Context: " + req.Context + "\n\n" + prompt
	changes = append(changes, "Added contextual background")
	return enhanced, changes
}

// applyClarityEnhancement improves prompt clarity and structure
func (r *Rewriter) applyClarityEnhancement(prompt string, req *OptimizationRequest) (string, []string) {
	changes := []string{}
	optimized := prompt

	// Add clear instruction prefix if missing
	if !strings.HasPrefix(strings.ToLower(prompt), "please") &&
	   !strings.HasPrefix(strings.ToLower(prompt), "provide") &&
	   !strings.HasPrefix(strings.ToLower(prompt), "explain") {
		optimized = "Please " + strings.ToLower(string(prompt[0])) + prompt[1:]
		changes = append(changes, "Added polite instruction prefix")
	}

	// Add response format guidance
	if !strings.Contains(strings.ToLower(prompt), "format") &&
	   !strings.Contains(strings.ToLower(prompt), "structure") {
		optimized += "\n\nPlease structure your response clearly with:\n" +
			"- A brief introduction\n" +
			"- Main points with explanations\n" +
			"- A concise summary"
		changes = append(changes, "Added response format guidance")
	}

	return optimized, changes
}

// applySpecificity makes the prompt more specific and detailed
func (r *Rewriter) applySpecificity(prompt string, req *OptimizationRequest) (string, []string) {
	changes := []string{}
	optimized := prompt

	// Add specificity qualifiers
	vaguePhrases := map[string]string{
		"tell me about":     "provide a comprehensive overview of",
		"explain":           "explain in detail, covering key concepts and examples for",
		"what is":           "define and explain the significance of",
		"how does":          "describe the mechanism and process of how",
	}

	for vague, specific := range vaguePhrases {
		if strings.Contains(strings.ToLower(optimized), vague) {
			optimized = strings.ReplaceAll(optimized, vague, specific)
			changes = append(changes, fmt.Sprintf("Enhanced specificity: '%s' → '%s'", vague, specific))
		}
	}

	// Add quality constraints
	if !strings.Contains(strings.ToLower(prompt), "accurate") &&
	   !strings.Contains(strings.ToLower(prompt), "verified") {
		optimized += "\n\nEnsure all information provided is accurate and well-supported."
		changes = append(changes, "Added accuracy constraints")
	}

	return optimized, changes
}

