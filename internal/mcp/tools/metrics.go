package tools

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MetricsTool handles metrics queries
type MetricsTool struct {
	k8sClient client.Client
}

// NewMetricsTool creates a new metrics tool
func NewMetricsTool(k8sClient client.Client) *MetricsTool {
	return &MetricsTool{
		k8sClient: k8sClient,
	}
}

// GetMetrics queries metrics for a customer
func (m *MetricsTool) GetMetrics(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	customerID, ok := params["customer_id"].(string)
	if !ok {
		return nil, fmt.Errorf("customer_id is required")
	}

	timeRange, ok := params["time_range"].(string)
	if !ok {
		timeRange = "24h"
	}

	// Parse requested metrics
	requestedMetrics := []string{"latency", "throughput"}
	if metrics, ok := params["metrics"].([]interface{}); ok {
		requestedMetrics = make([]string, 0, len(metrics))
		for _, m := range metrics {
			if metricName, ok := m.(string); ok {
				requestedMetrics = append(requestedMetrics, metricName)
			}
		}
	}

	// In a real implementation, this would query Prometheus
	// For now, we'll return simulated data
	metrics := make(map[string]interface{})

	for _, metricName := range requestedMetrics {
		switch metricName {
		case "latency":
			metrics["latency_p50"] = 250.0 // milliseconds
			metrics["latency_p95"] = 450.0
			metrics["latency_p99"] = 580.0
		case "throughput":
			metrics["requests_per_second"] = 42.5
			metrics["tokens_per_second"] = 1250.0
		case "error_rate":
			metrics["error_rate"] = 0.02 // 2%
		case "gpu_utilization":
			metrics["gpu_utilization"] = 0.75 // 75%
		}
	}

	return map[string]interface{}{
		"customer_id": customerID,
		"time_range":  timeRange,
		"metrics":     metrics,
		"timestamp":   time.Now().Format(time.RFC3339),
	}, nil
}

// DetectAnomalies detects anomalies in metrics using z-score
func (m *MetricsTool) DetectAnomalies(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	metricName, ok := params["metric_name"].(string)
	if !ok {
		return nil, fmt.Errorf("metric_name is required")
	}

	threshold := 3.0 // Default z-score threshold
	if t, ok := params["threshold"].(float64); ok {
		threshold = t
	}

	timeWindow, ok := params["time_window"].(string)
	if !ok {
		timeWindow = "24h"
	}

	// In a real implementation, this would query historical data
	// For now, we'll simulate anomaly detection
	anomalies := m.simulateAnomalyDetection(metricName, threshold)

	return map[string]interface{}{
		"metric_name": metricName,
		"time_window": timeWindow,
		"threshold":   threshold,
		"anomalies":   anomalies,
		"count":       len(anomalies),
		"analyzed_at": time.Now().Format(time.RFC3339),
	}, nil
}

// simulateAnomalyDetection simulates anomaly detection for demo purposes
func (m *MetricsTool) simulateAnomalyDetection(metricName string, threshold float64) []map[string]interface{} {
	anomalies := make([]map[string]interface{}, 0)

	// Generate some sample data with occasional anomalies
	now := time.Now()
	baseline := 250.0 // baseline value (e.g., 250ms latency)
	stdDev := 50.0    // standard deviation

	for i := 0; i < 100; i++ {
		timestamp := now.Add(time.Duration(-i) * time.Minute)
		
		// Generate value with occasional spikes
		value := baseline + (rand.Float64()-0.5)*2*stdDev
		if rand.Float64() < 0.05 { // 5% chance of anomaly
			value = baseline + stdDev*threshold*1.5 // Create an anomaly
		}

		// Calculate z-score
		zScore := math.Abs((value - baseline) / stdDev)

		if zScore > threshold {
			anomalies = append(anomalies, map[string]interface{}{
				"timestamp":  timestamp.Format(time.RFC3339),
				"value":      value,
				"baseline":   baseline,
				"z_score":    zScore,
				"deviation":  fmt.Sprintf("%.1f%%", ((value-baseline)/baseline)*100),
			})
		}
	}

	return anomalies
}

