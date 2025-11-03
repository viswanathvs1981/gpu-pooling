"""
Workflow Bridge between Microsoft Agent Framework and NexusAI Go Agents

Allows Microsoft Framework workflows to trigger and interact with Go agents.
"""

import httpx
import asyncio
from typing import Dict, Any, Optional
from .config import NexusAIConfig


class WorkflowBridge:
    """
    Bridge for Microsoft Agent Framework workflows to interact with
    NexusAI Go-based infrastructure agents.
    """
    
    def __init__(self, config: Optional[NexusAIConfig] = None):
        self.config = config or NexusAIConfig.from_env()
        self.client = httpx.AsyncClient(timeout=60.0)
    
    # ========== Go Agent Triggers ==========
    
    async def trigger_go_orchestrator(
        self,
        intent: str,
        params: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Trigger Go orchestrator for simple workflows"""
        response = await self.client.post(
            "http://orchestrator:8080/api/v1/requests",
            json={
                "request": intent,
                "params": params
            }
        )
        response.raise_for_status()
        return response.json()
    
    async def notify_resource_agent(
        self,
        event_type: str,
        data: Dict[str, Any]
    ) -> None:
        """Notify Go Resource Agent of events"""
        # Resource Agent listens on Redis, so publish event
        # In real implementation, would use Redis client
        pass
    
    async def get_analytics(
        self,
        metric_name: str,
        time_range: str = "1h"
    ) -> Dict[str, Any]:
        """Get analytics from Go Analytics Agent"""
        response = await self.client.get(
            f"http://analytics-agent:8081/api/v1/metrics",
            params={
                "metric": metric_name,
                "range": time_range
            }
        )
        response.raise_for_status()
        return response.json()
    
    # ========== Safety Integration ==========
    
    async def check_safety(
        self,
        text: str,
        context: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Check AI safety before processing"""
        response = await self.client.post(
            f"{self.config.safety_endpoint}/v1/check-safety",
            json={
                "text": text,
                "context": context or {}
            }
        )
        response.raise_for_status()
        return response.json()
    
    async def optimize_prompt(
        self,
        prompt: str,
        optimize_tokens: bool = True,
        max_tokens: Optional[int] = None
    ) -> Dict[str, Any]:
        """Optimize prompt before sending to LLM"""
        response = await self.client.post(
            f"{self.config.prompt_optimizer_endpoint}/v1/optimize",
            json={
                "original_prompt": prompt,
                "optimize_tokens": optimize_tokens,
                "max_tokens": max_tokens
            }
        )
        response.raise_for_status()
        return response.json()
    
    # ========== Checkpointing Support ==========
    
    async def save_workflow_checkpoint(
        self,
        workflow_id: str,
        state: Dict[str, Any]
    ) -> None:
        """Save workflow checkpoint to platform storage"""
        # Store in GreptimeDB for durability
        response = await self.client.post(
            "http://greptime-service:4000/v1/sql",
            json={
                "sql": f"""
                    INSERT INTO workflow_checkpoints (
                        workflow_id, state, timestamp
                    ) VALUES (
                        '{workflow_id}',
                        '{str(state)}',
                        NOW()
                    )
                """
            }
        )
        response.raise_for_status()
    
    async def load_workflow_checkpoint(
        self,
        workflow_id: str
    ) -> Optional[Dict[str, Any]]:
        """Load workflow checkpoint from storage"""
        response = await self.client.post(
            "http://greptime-service:4000/v1/sql",
            json={
                "sql": f"""
                    SELECT state FROM workflow_checkpoints
                    WHERE workflow_id = '{workflow_id}'
                    ORDER BY timestamp DESC
                    LIMIT 1
                """
            }
        )
        response.raise_for_status()
        result = response.json()
        
        if result.get("rows"):
            return eval(result["rows"][0][0])  # Convert string back to dict
        return None
    
    async def close(self):
        """Close HTTP client"""
        await self.client.aclose()

