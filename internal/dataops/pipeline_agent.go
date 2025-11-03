package dataops

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
)

// DataPipelineAgent handles schema inference and auto-healing
type DataPipelineAgent struct {
	schemaInferrer *SchemaInferrer
	qualityChecker *DataQualityChecker
}

// SchemaInferrer infers schema from data
type SchemaInferrer struct{}

// DataQualityChecker checks data quality
type DataQualityChecker struct{}

// FieldSchema represents a field's schema
type FieldSchema struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Format   string `json:"format,omitempty"`
}

// SchemaInferenceRequest represents schema inference request
type SchemaInferenceRequest struct {
	Data        []map[string]interface{} `json:"data"`
	SampleSize  int                      `json:"sample_size,omitempty"`
}

// SchemaInferenceResponse represents inferred schema
type SchemaInferenceResponse struct {
	Schema  []FieldSchema `json:"schema"`
	Quality float64       `json:"quality"`
	Issues  []string      `json:"issues,omitempty"`
}

// NewDataPipelineAgent creates a new data pipeline agent
func NewDataPipelineAgent() *DataPipelineAgent {
	return &DataPipelineAgent{
		schemaInferrer: &SchemaInferrer{},
		qualityChecker: &DataQualityChecker{},
	}
}

// InferSchema infers schema from data samples
func (s *SchemaInferrer) InferSchema(data []map[string]interface{}) []FieldSchema {
	if len(data) == 0 {
		return nil
	}

	schema := []FieldSchema{}
	fieldTypes := make(map[string]map[string]int)
	
	// Analyze all records
	for _, record := range data {
		for fieldName, value := range record {
			if _, exists := fieldTypes[fieldName]; !exists {
				fieldTypes[fieldName] = make(map[string]int)
			}
			
			detectedType := s.detectType(value)
			fieldTypes[fieldName][detectedType]++
		}
	}

	// Determine most common type for each field
	for fieldName, types := range fieldTypes {
		mostCommonType := ""
		maxCount := 0
		totalCount := 0
		
		for typeName, count := range types {
			totalCount += count
			if count > maxCount {
				maxCount = count
				mostCommonType = typeName
			}
		}
		
		// Field is nullable if it doesn't appear in all records
		nullable := totalCount < len(data)
		
		// Detect format (e.g., email, phone, date)
		format := s.detectFormat(fieldName, mostCommonType, data)
		
		schema = append(schema, FieldSchema{
			Name:     fieldName,
			Type:     mostCommonType,
			Nullable: nullable,
			Format:   format,
		})
	}

	return schema
}

// detectType detects the type of a value
func (s *SchemaInferrer) detectType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case bool:
		return "boolean"
	case float64, int, int64:
		return "number"
	case string:
		// Check if it's a date, email, or other special format
		if s.isDate(v) {
			return "date"
		}
		if s.isEmail(v) {
			return "email"
		}
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}

// detectFormat detects special formats
func (s *SchemaInferrer) detectFormat(fieldName, fieldType string, data []map[string]interface{}) string {
	if fieldType != "string" {
		return ""
	}

	// Sample a few values
	sampleSize := 10
	if len(data) < sampleSize {
		sampleSize = len(data)
	}

	emailCount := 0
	phoneCount := 0
	dateCount := 0

	for i := 0; i < sampleSize; i++ {
		if value, ok := data[i][fieldName]; ok {
			if strValue, ok := value.(string); ok {
				if s.isEmail(strValue) {
					emailCount++
				}
				if s.isPhone(strValue) {
					phoneCount++
				}
				if s.isDate(strValue) {
					dateCount++
				}
			}
		}
	}

	// If >80% of samples match a format, use that format
	threshold := int(float64(sampleSize) * 0.8)
	if emailCount >= threshold {
		return "email"
	}
	if phoneCount >= threshold {
		return "phone"
	}
	if dateCount >= threshold {
		return "date"
	}

	return ""
}

// isEmail checks if string is an email
func (s *SchemaInferrer) isEmail(value string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(value)
}

// isPhone checks if string is a phone number
func (s *SchemaInferrer) isPhone(value string) bool {
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	cleanedValue := regexp.MustCompile(`[^0-9+]`).ReplaceAllString(value, "")
	return phoneRegex.MatchString(cleanedValue)
}

// isDate checks if string is a date
func (s *SchemaInferrer) isDate(value string) bool {
	datePatterns := []string{
		`^\d{4}-\d{2}-\d{2}$`,                    // YYYY-MM-DD
		`^\d{2}/\d{2}/\d{4}$`,                    // MM/DD/YYYY
		`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`,   // ISO 8601
	}

	for _, pattern := range datePatterns {
		if matched, _ := regexp.MatchString(pattern, value); matched {
			return true
		}
	}

	return false
}

// CheckQuality performs data quality checks
func (q *DataQualityChecker) CheckQuality(data []map[string]interface{}, schema []FieldSchema) (float64, []string) {
	if len(data) == 0 {
		return 0.0, []string{"No data provided"}
	}

	issues := []string{}
	totalChecks := 0
	passedChecks := 0

	// Build schema map for quick lookup
	schemaMap := make(map[string]FieldSchema)
	for _, field := range schema {
		schemaMap[field.Name] = field
	}

	// Check each record
	for i, record := range data {
		for fieldName, expectedSchema := range schemaMap {
			totalChecks++
			
			value, exists := record[fieldName]
			
			// Check nullability
			if !exists || value == nil {
				if !expectedSchema.Nullable {
					issues = append(issues, fmt.Sprintf("Record %d: missing required field '%s'", i, fieldName))
				} else {
					passedChecks++
				}
				continue
			}

			// Check type
			actualType := (&SchemaInferrer{}).detectType(value)
			if actualType != expectedSchema.Type {
				issues = append(issues, fmt.Sprintf("Record %d: field '%s' has type '%s', expected '%s'", 
					i, fieldName, actualType, expectedSchema.Type))
			} else {
				passedChecks++
			}

			// Check format
			if expectedSchema.Format != "" && actualType == "string" {
				strValue := value.(string)
				validFormat := false

				switch expectedSchema.Format {
				case "email":
					validFormat = (&SchemaInferrer{}).isEmail(strValue)
				case "phone":
					validFormat = (&SchemaInferrer{}).isPhone(strValue)
				case "date":
					validFormat = (&SchemaInferrer{}).isDate(strValue)
				default:
					validFormat = true
				}

				if !validFormat {
					issues = append(issues, fmt.Sprintf("Record %d: field '%s' has invalid format (expected %s)", 
						i, fieldName, expectedSchema.Format))
				}
			}
		}
	}

	quality := float64(passedChecks) / float64(totalChecks)
	return quality, issues
}

// HTTPHandler provides HTTP endpoints
func (a *DataPipelineAgent) HTTPHandler() http.Handler {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/v1/infer-schema", a.handleInferSchema)
	mux.HandleFunc("/v1/check-quality", a.handleCheckQuality)
	mux.HandleFunc("/health", a.handleHealth)
	
	return mux
}

func (a *DataPipelineAgent) handleInferSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SchemaInferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	schema := a.schemaInferrer.InferSchema(req.Data)
	quality, issues := a.qualityChecker.CheckQuality(req.Data, schema)

	resp := SchemaInferenceResponse{
		Schema:  schema,
		Quality: quality,
		Issues:  issues,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *DataPipelineAgent) handleCheckQuality(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Data   []map[string]interface{} `json:"data"`
		Schema []FieldSchema            `json:"schema"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	quality, issues := a.qualityChecker.CheckQuality(req.Data, req.Schema)

	resp := map[string]interface{}{
		"quality": quality,
		"issues":  issues,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (a *DataPipelineAgent) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Start starts the data pipeline agent
func (a *DataPipelineAgent) Start(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: a.HTTPHandler(),
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("Data Pipeline Agent listening on %s", addr)
	return server.ListenAndServe()
}

