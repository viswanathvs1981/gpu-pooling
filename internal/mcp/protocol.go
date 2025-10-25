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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Handler     ToolHandler            `json:"-"`
}

// ToolHandler is a function that handles tool execution
type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

// ToolRequest represents a tool execution request
type ToolRequest struct {
	ToolName string                 `json:"tool_name"`
	Params   map[string]interface{} `json:"params"`
}

// ToolResponse represents a tool execution response
type ToolResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Server represents an MCP server that provides tools
type Server struct {
	Name        string
	Description string
	Tools       map[string]*Tool
}

// NewServer creates a new MCP server
func NewServer(name, description string) *Server {
	return &Server{
		Name:        name,
		Description: description,
		Tools:       make(map[string]*Tool),
	}
}

// RegisterTool registers a new tool
func (s *Server) RegisterTool(tool *Tool) {
	s.Tools[tool.Name] = tool
}

// ExecuteTool executes a tool by name
func (s *Server) ExecuteTool(ctx context.Context, req *ToolRequest) (*ToolResponse, error) {
	tool, exists := s.Tools[req.ToolName]
	if !exists {
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("tool not found: %s", req.ToolName),
		}, nil
	}

	result, err := tool.Handler(ctx, req.Params)
	if err != nil {
		return &ToolResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResponse{
		Success: true,
		Result:  result,
	}, nil
}

// ListTools returns all available tools
func (s *Server) ListTools() []*Tool {
	tools := make([]*Tool, 0, len(s.Tools))
	for _, tool := range s.Tools {
		tools = append(tools, tool)
	}
	return tools
}

// Gateway manages multiple MCP servers
type Gateway struct {
	Servers map[string]*Server
}

// NewGateway creates a new MCP gateway
func NewGateway() *Gateway {
	return &Gateway{
		Servers: make(map[string]*Server),
	}
}

// RegisterServer registers an MCP server
func (g *Gateway) RegisterServer(server *Server) {
	g.Servers[server.Name] = server
}

// ExecuteTool routes tool execution to the appropriate server
func (g *Gateway) ExecuteTool(ctx context.Context, serverName string, req *ToolRequest) (*ToolResponse, error) {
	server, exists := g.Servers[serverName]
	if !exists {
		return &ToolResponse{
			Success: false,
			Error:   fmt.Sprintf("server not found: %s", serverName),
		}, nil
	}

	return server.ExecuteTool(ctx, req)
}

// ListServers returns all registered servers
func (g *Gateway) ListServers() []*Server {
	servers := make([]*Server, 0, len(g.Servers))
	for _, server := range g.Servers {
		servers = append(servers, server)
	}
	return servers
}

// GetServerTools returns all tools from a specific server
func (g *Gateway) GetServerTools(serverName string) ([]*Tool, error) {
	server, exists := g.Servers[serverName]
	if !exists {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}

	return server.ListTools(), nil
}

// Helper function to create tool schemas
func CreateToolSchema(required []string, properties map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"required":   required,
		"properties": properties,
	}
}

// Helper function to marshal tool response to JSON
func (tr *ToolResponse) ToJSON() ([]byte, error) {
	return json.Marshal(tr)
}



