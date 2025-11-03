"""
NexusAI Configuration for Microsoft Agent Framework
"""

import os
from dataclasses import dataclass
from typing import Optional


@dataclass
class NexusAIConfig:
    """Configuration for NexusAI platform integration"""
    
    # MCP Server
    mcp_url: str = "http://mcp-server:8080"
    
    # LLM Endpoints
    llm_endpoint: str = "http://vllm-service/v1/chat/completions"
    
    # Memory Service
    memory_endpoint: str = "http://memory-service:8090"
    
    # AI Safety Service
    safety_endpoint: str = "http://aisafety-service/safety"
    
    # Prompt Optimizer
    prompt_optimizer_endpoint: str = "http://prompt-optimizer:8888"
    
    # Resource Allocation
    gpu_quota: float = 1.0  # vGPU allocation (0.1 - 10.0)
    
    # Cost Tracking
    cost_tracking: bool = True
    tenant_id: Optional[str] = None
    
    # Observability
    enable_telemetry: bool = True
    
    @classmethod
    def from_env(cls) -> "NexusAIConfig":
        """Create configuration from environment variables"""
        return cls(
            mcp_url=os.getenv("NEXUSAI_MCP_URL", "http://mcp-server:8080"),
            llm_endpoint=os.getenv("NEXUSAI_LLM_ENDPOINT", "http://vllm-service/v1/chat/completions"),
            memory_endpoint=os.getenv("NEXUSAI_MEMORY_ENDPOINT", "http://memory-service:8090"),
            safety_endpoint=os.getenv("NEXUSAI_SAFETY_ENDPOINT", "http://aisafety-service/safety"),
            prompt_optimizer_endpoint=os.getenv("NEXUSAI_PROMPT_OPTIMIZER", "http://prompt-optimizer:8888"),
            gpu_quota=float(os.getenv("NEXUSAI_GPU_QUOTA", "1.0")),
            cost_tracking=os.getenv("NEXUSAI_COST_TRACKING", "true").lower() == "true",
            tenant_id=os.getenv("NEXUSAI_TENANT_ID"),
            enable_telemetry=os.getenv("NEXUSAI_TELEMETRY", "true").lower() == "true",
        )

