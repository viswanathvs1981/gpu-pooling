package tools

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CostsTool handles cost analysis and optimization
type CostsTool struct {
	k8sClient client.Client
}

// NewCostsTool creates a new costs tool
func NewCostsTool(k8sClient client.Client) *CostsTool {
	return &CostsTool{
		k8sClient: k8sClient,
	}
}

// GetCosts returns cost breakdown for a customer
func (c *CostsTool) GetCosts(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	customerID, ok := params["customer_id"].(string)
	if !ok {
		return nil, fmt.Errorf("customer_id is required")
	}

	period, ok := params["period"].(map[string]interface{})
	if !ok {
		// Default to last 30 days
		now := time.Now()
		period = map[string]interface{}{
			"start": now.AddDate(0, 0, -30).Format(time.RFC3339),
			"end":   now.Format(time.RFC3339),
		}
	}

	// In a real implementation, this would query GreptimeDB or similar
	// For now, we'll return simulated cost data
	gpuCost := 1584.0      // $1,584/month for GPU usage
	inferenceCost := 256.0 // $256/month for inference
	trainingCost := 450.0  // $450/month for training
	totalCost := gpuCost + inferenceCost + trainingCost

	return map[string]interface{}{
		"customer_id": customerID,
		"period":      period,
		"total_cost":  totalCost,
		"breakdown": map[string]interface{}{
			"gpu_compute":    gpuCost,
			"inference_api":  inferenceCost,
			"training_jobs":  trainingCost,
			"storage":        24.0,
			"network":        12.0,
		},
		"usage_stats": map[string]interface{}{
			"vgpu_hours":        660.0,
			"inference_requests": 1250000,
			"training_jobs":      8,
			"tokens_processed":  42500000,
		},
		"currency":   "USD",
		"calculated_at": time.Now().Format(time.RFC3339),
	}, nil
}

// QueryUsage queries usage patterns
func (c *CostsTool) QueryUsage(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	filters, ok := params["filters"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("filters are required")
	}

	customerID, _ := filters["customer"].(string)
	timeRange, ok := filters["time_range"].(string)
	if !ok {
		timeRange = "7d"
	}

	// Simulate usage pattern analysis
	patterns := c.analyzeUsagePatterns(customerID, timeRange)

	return map[string]interface{}{
		"customer_id":  customerID,
		"time_range":   timeRange,
		"request_count": 125000,
		"token_count":   4250000,
		"avg_latency":   320.0, // milliseconds
		"patterns":      patterns,
		"analyzed_at":   time.Now().Format(time.RFC3339),
	}, nil
}

// ForecastCosts forecasts future costs based on historical data
func (c *CostsTool) ForecastCosts(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	customerID, ok := params["customer_id"].(string)
	if !ok {
		return nil, fmt.Errorf("customer_id is required")
	}

	forecastDays := 30
	if days, ok := params["forecast_days"].(float64); ok {
		forecastDays = int(days)
	}

	// Simple linear forecast (in reality, use time series forecasting)
	currentDailyCost := 75.0 // $75/day current
	growthRate := 0.05       // 5% growth rate

	forecast := make([]map[string]interface{}, forecastDays)
	for i := 0; i < forecastDays; i++ {
		date := time.Now().AddDate(0, 0, i+1)
		estimatedCost := currentDailyCost * (1 + float64(i)*growthRate/30)
		
		forecast[i] = map[string]interface{}{
			"date":           date.Format("2006-01-02"),
			"estimated_cost": estimatedCost,
			"confidence":     0.85, // 85% confidence
		}
	}

	totalForecast := 0.0
	for _, day := range forecast {
		totalForecast += day["estimated_cost"].(float64)
	}

	return map[string]interface{}{
		"customer_id":     customerID,
		"forecast_days":   forecastDays,
		"forecast":        forecast,
		"total_forecast":  totalForecast,
		"confidence_level": 0.85,
		"method":          "linear_regression",
		"generated_at":    time.Now().Format(time.RFC3339),
	}, nil
}

// RecommendOptimization recommends cost/performance optimizations
func (c *CostsTool) RecommendOptimization(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	customerID, ok := params["customer_id"].(string)
	if !ok {
		return nil, fmt.Errorf("customer_id is required")
	}

	target := "cost"
	if t, ok := params["optimization_target"].(string); ok {
		target = t
	}

	// Generate recommendations based on target
	recommendations := c.generateRecommendations(customerID, target)

	return map[string]interface{}{
		"customer_id":         customerID,
		"optimization_target": target,
		"recommendations":     recommendations,
		"total_potential_savings": c.calculateTotalSavings(recommendations),
		"generated_at":        time.Now().Format(time.RFC3339),
	}, nil
}

// analyzeUsagePatterns analyzes usage patterns
func (c *CostsTool) analyzeUsagePatterns(customerID, timeRange string) map[string]interface{} {
	return map[string]interface{}{
		"peak_hours": []int{9, 10, 11, 14, 15, 16}, // 9am-11am, 2pm-4pm
		"request_distribution": map[string]interface{}{
			"short_requests": 0.30,  // < 500 tokens
			"medium_requests": 0.50, // 500-2000 tokens
			"long_requests": 0.20,   // > 2000 tokens
		},
		"model_usage": map[string]interface{}{
			"gpt-4": 0.40,
			"llama-3-8b": 0.45,
			"mistral-7b": 0.15,
		},
		"gpu_utilization": map[string]interface{}{
			"average": 0.68,
			"peak": 0.92,
			"off_peak": 0.35,
		},
	}
}

// generateRecommendations generates optimization recommendations
func (c *CostsTool) generateRecommendations(customerID, target string) []map[string]interface{} {
	recommendations := make([]map[string]interface{}, 0)

	if target == "cost" {
		recommendations = append(recommendations, map[string]interface{}{
			"type":        "routing_optimization",
			"description": "Route 40% of requests (>2000 tokens) to self-hosted instead of Azure",
			"current_cost": 320.0,
			"optimized_cost": 192.0,
			"savings":     128.0,
			"savings_pct": 40.0,
			"effort":      "low",
			"implementation": "Update LLMRoute configuration",
		})

		recommendations = append(recommendations, map[string]interface{}{
			"type":        "gpu_rightsizing",
			"description": "Scale down GPU allocation during off-peak hours (8pm-6am)",
			"current_cost": 480.0,
			"optimized_cost": 360.0,
			"savings":     120.0,
			"savings_pct": 25.0,
			"effort":      "medium",
			"implementation": "Configure autoscaler schedule",
		})

		recommendations = append(recommendations, map[string]interface{}{
			"type":        "spot_instances",
			"description": "Use spot instances for non-critical inference workloads",
			"current_cost": 240.0,
			"optimized_cost": 96.0,
			"savings":     144.0,
			"savings_pct": 60.0,
			"effort":      "medium",
			"implementation": "Enable spot instance pool",
		})
	} else if target == "latency" {
		recommendations = append(recommendations, map[string]interface{}{
			"type":        "increase_replicas",
			"description": "Increase vLLM replicas from 1 to 2 during peak hours",
			"current_latency": 450.0, // p99 ms
			"optimized_latency": 280.0,
			"improvement": "37% faster",
			"cost_impact": 360.0, // Additional monthly cost
			"implementation": "Update Deployment replicas",
		})
	}

	return recommendations
}

// calculateTotalSavings calculates total potential savings
func (c *CostsTool) calculateTotalSavings(recommendations []map[string]interface{}) float64 {
	total := 0.0
	for _, rec := range recommendations {
		if savings, ok := rec["savings"].(float64); ok {
			total += savings
		}
	}
	return total
}

