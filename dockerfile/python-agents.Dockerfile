# Multi-stage Dockerfile for NexusAI Python Agents (Microsoft Agent Framework)
# Supports: Orchestrator, Training, Deployment, Cost, SmallModel, Hybrid agents

FROM python:3.11-slim AS builder

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    make \
    && rm -rf /var/lib/apt/lists/*

# Copy SDK and requirements
COPY python-sdk /build/python-sdk
COPY agents /build/agents

# Install Python dependencies
# Note: asyncio is built-in to Python 3.11, no need to install separately
# pydantic>=2.11.3 required by agent-framework-a2a
# redis>=6.4.0 required by agent-framework-redis
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir \
    agent-framework==1.0.0b251028 \
    "pydantic>=2.11.3" \
    "redis>=6.4.0" \
    && cd /build/python-sdk && pip install -e .

# ===== Runtime Stage =====
FROM python:3.11-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy Python packages from builder
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Copy agents
COPY agents /app/agents
COPY python-sdk /app/python-sdk

# Create non-root user
RUN useradd -m -u 1000 -s /bin/bash nexusai && \
    chown -R nexusai:nexusai /app

USER nexusai

# Default environment variables
ENV PYTHONUNBUFFERED=1 \
    NEXUSAI_MCP_URL=http://mcp-server:8080 \
    NEXUSAI_LLM_ENDPOINT=http://vllm-service/v1/chat/completions \
    NEXUSAI_MEMORY_ENDPOINT=http://memory-service:8090 \
    NEXUSAI_SAFETY_ENDPOINT=http://aisafety-service/safety \
    NEXUSAI_PROMPT_OPTIMIZER=http://prompt-optimizer:8888

# Agent type is specified via CMD
ENTRYPOINT ["python", "-m"]

# Default: Orchestrator agent
CMD ["agents.orchestrator_agent"]

