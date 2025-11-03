"""
NexusAI Platform SDK for Microsoft Agent Framework Integration

Provides tools and utilities for building agents on NexusAI infrastructure.
"""

from .mcp_tools import NexusAIMCPTools
from .config import NexusAIConfig
from .workflow_bridge import WorkflowBridge

__version__ = "1.0.0"

__all__ = [
    "NexusAIMCPTools",
    "NexusAIConfig",
    "WorkflowBridge",
]

