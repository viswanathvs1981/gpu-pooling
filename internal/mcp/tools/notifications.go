/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/NexusGPU/tensor-fusion/internal/mcp"
)

// CreateNotificationServer creates the notification MCP server with all tools
func CreateNotificationServer() *mcp.Server {
	server := mcp.NewServer("slack", "Notification and Alerting Tools")

	// send_message tool
	server.RegisterTool(&mcp.Tool{
		Name:        "send_message",
		Description: "Send a message to Slack channel",
		InputSchema: mcp.CreateToolSchema(
			[]string{"channel", "message"},
			map[string]interface{}{
				"channel": map[string]string{
					"type":        "string",
					"description": "Slack channel name (e.g., '#deployments', '@user')",
				},
				"message": map[string]string{
					"type":        "string",
					"description": "Message content",
				},
				"priority": map[string]string{
					"type":        "string",
					"description": "Message priority (low, normal, high, urgent)",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			channel := params["channel"].(string)
			message := params["message"].(string)

			// In real implementation, send to Slack API
			messageID := fmt.Sprintf("msg-%d", time.Now().Unix())

			return map[string]interface{}{
				"message_id": messageID,
				"channel":    channel,
				"timestamp":  time.Now().Format(time.RFC3339),
				"status":     "sent",
			}, nil
		},
	})

	// send_email tool
	server.RegisterTool(&mcp.Tool{
		Name:        "send_email",
		Description: "Send an email notification",
		InputSchema: mcp.CreateToolSchema(
			[]string{"to", "subject", "body"},
			map[string]interface{}{
				"to": map[string]string{
					"type":        "string",
					"description": "Recipient email address",
				},
				"subject": map[string]string{
					"type":        "string",
					"description": "Email subject",
				},
				"body": map[string]string{
					"type":        "string",
					"description": "Email body (HTML supported)",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			to := params["to"].(string)
			subject := params["subject"].(string)

			return map[string]interface{}{
				"email_id":  fmt.Sprintf("email-%d", time.Now().Unix()),
				"to":        to,
				"subject":   subject,
				"timestamp": time.Now().Format(time.RFC3339),
				"status":    "sent",
			}, nil
		},
	})

	// create_alert tool
	server.RegisterTool(&mcp.Tool{
		Name:        "create_alert",
		Description: "Create an alert in the monitoring system",
		InputSchema: mcp.CreateToolSchema(
			[]string{"title", "severity"},
			map[string]interface{}{
				"title": map[string]string{
					"type":        "string",
					"description": "Alert title",
				},
				"severity": map[string]string{
					"type":        "string",
					"description": "Alert severity (info, warning, error, critical)",
				},
				"description": map[string]string{
					"type":        "string",
					"description": "Alert description",
				},
				"labels": map[string]interface{}{
					"type":        "object",
					"description": "Alert labels",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			title := params["title"].(string)
			severity := params["severity"].(string)

			return map[string]interface{}{
				"alert_id":  fmt.Sprintf("alert-%d", time.Now().Unix()),
				"title":     title,
				"severity":  severity,
				"timestamp": time.Now().Format(time.RFC3339),
				"status":    "active",
			}, nil
		},
	})

	// send_webhook tool
	server.RegisterTool(&mcp.Tool{
		Name:        "send_webhook",
		Description: "Send a webhook notification",
		InputSchema: mcp.CreateToolSchema(
			[]string{"url", "payload"},
			map[string]interface{}{
				"url": map[string]string{
					"type":        "string",
					"description": "Webhook URL",
				},
				"payload": map[string]interface{}{
					"type":        "object",
					"description": "Webhook payload (JSON)",
				},
				"method": map[string]string{
					"type":        "string",
					"description": "HTTP method (POST, PUT)",
				},
			},
		),
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			url := params["url"].(string)

			return map[string]interface{}{
				"webhook_id": fmt.Sprintf("webhook-%d", time.Now().Unix()),
				"url":        url,
				"timestamp":  time.Now().Format(time.RFC3339),
				"status":     "sent",
				"response_code": 200,
			}, nil
		},
	})

	return server
}



