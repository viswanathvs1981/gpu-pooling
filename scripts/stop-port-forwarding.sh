#!/bin/bash

# NexusAI Platform - Stop Port Forwarding Script

echo "🛑 Stopping all port forwarding processes..."

# Kill all kubectl port-forward processes
pkill -f "kubectl port-forward" || echo "No port-forward processes found"

# Clean up PID files
rm -f /tmp/port-forward-*.pid

echo "✅ All port forwarding stopped"

