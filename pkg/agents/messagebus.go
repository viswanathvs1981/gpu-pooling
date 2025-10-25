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

package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// MessageBus handles Agent-to-Agent communication
type MessageBus struct {
	redisClient *redis.Client
	subscribers map[string]chan *Message
}

// NewMessageBus creates a new message bus using Redis Pub/Sub
func NewMessageBus(redisAddr string) (*MessageBus, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &MessageBus{
		redisClient: client,
		subscribers: make(map[string]chan *Message),
	}, nil
}

// Publish sends a message to the specified agent
func (mb *MessageBus) Publish(ctx context.Context, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	channel := fmt.Sprintf("agent:%s", msg.To)
	if err := mb.redisClient.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Subscribe subscribes an agent to receive messages
func (mb *MessageBus) Subscribe(ctx context.Context, agentID string, handler func(*Message)) error {
	channel := fmt.Sprintf("agent:%s", agentID)
	pubsub := mb.redisClient.Subscribe(ctx, channel)

	// Wait for confirmation
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Start message handler goroutine
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			var agentMsg Message
			if err := json.Unmarshal([]byte(msg.Payload), &agentMsg); err != nil {
				continue
			}
			handler(&agentMsg)
		}
	}()

	return nil
}

// Broadcast sends a message to all agents
func (mb *MessageBus) Broadcast(ctx context.Context, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := mb.redisClient.Publish(ctx, "agent:broadcast", data).Err(); err != nil {
		return fmt.Errorf("failed to broadcast message: %w", err)
	}

	return nil
}

// Request sends a request and waits for a response
func (mb *MessageBus) Request(ctx context.Context, msg *Message, timeout time.Duration) (*Message, error) {
	// Create response channel
	responseChannel := fmt.Sprintf("agent:%s:response:%d", msg.From, time.Now().UnixNano())
	msg.Params["_responseChannel"] = responseChannel

	// Subscribe to response channel
	pubsub := mb.redisClient.Subscribe(ctx, responseChannel)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to response channel: %w", err)
	}

	// Publish request
	if err := mb.Publish(ctx, msg); err != nil {
		return nil, err
	}

	// Wait for response with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch := pubsub.Channel()
	select {
	case responseMsg := <-ch:
		var response Message
		if err := json.Unmarshal([]byte(responseMsg.Payload), &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		return &response, nil
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("request timeout after %v", timeout)
	}
}

// SendResponse sends a response to a request
func (mb *MessageBus) SendResponse(ctx context.Context, originalMsg *Message, result interface{}, err error) error {
	responseChannel, ok := originalMsg.Params["_responseChannel"].(string)
	if !ok {
		return fmt.Errorf("no response channel in request")
	}

	response := &Message{
		From:      originalMsg.To,
		To:        originalMsg.From,
		Type:      "response",
		Method:    originalMsg.Method,
		Result:    result,
		Timestamp: time.Now(),
	}

	if err != nil {
		response.Error = err.Error()
	}

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := mb.redisClient.Publish(ctx, responseChannel, data).Err(); err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	return nil
}

// Close closes the message bus connection
func (mb *MessageBus) Close() error {
	return mb.redisClient.Close()
}

