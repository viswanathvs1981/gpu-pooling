"""
NexusAI MCP Tools - Platform operations for Microsoft Agent Framework
"""

import httpx
import json
from typing import Dict, Any, Optional, List
from .config import NexusAIConfig


class NexusAIMCPTools:
    """
    Wrapper for NexusAI MCP platform tools.
    Provides 20 operations for agent workflows.
    """
    
    def __init__(self, config: Optional[NexusAIConfig] = None):
        self.config = config or NexusAIConfig.from_env()
        self.client = httpx.AsyncClient(timeout=30.0)
    
    async def _call_mcp(self, method: str, params: Dict[str, Any]) -> Dict[str, Any]:
        """Call MCP tool via JSON-RPC 2.0"""
        request = {
            "jsonrpc": "2.0",
            "method": method,
            "params": params,
            "id": 1
        }
        
        response = await self.client.post(
            f"{self.config.mcp_url}/execute",
            json=request,
            headers={"Content-Type": "application/json"}
        )
        response.raise_for_status()
        
        result = response.json()
        if "error" in result:
            raise Exception(f"MCP Error: {result['error']['message']}")
        
        return result.get("result", {})
    
    # ========== Deployment Tools (4) ==========
    
    async def deploy_model(
        self,
        model_name: str,
        model_path: str,
        vgpu_size: float = 1.0,
        replicas: int = 1,
        **kwargs
    ) -> Dict[str, Any]:
        """Deploy LLM model to the platform"""
        params = {
            "model_name": model_name,
            "model_path": model_path,
            "vgpu_size": vgpu_size,
            "replicas": replicas,
            **kwargs
        }
        return await self._call_mcp("deploy_model", params)
    
    async def allocate_gpu(
        self,
        vgpu_size: float,
        duration: str = "24h",
        pool_name: str = "default-pool"
    ) -> Dict[str, Any]:
        """Allocate GPU resources"""
        params = {
            "vgpu_size": vgpu_size,
            "duration": duration,
            "pool_name": pool_name
        }
        return await self._call_mcp("allocate_gpu", params)
    
    async def update_routing(
        self,
        model_name: str,
        strategy: str = "cost-optimized",
        weights: Optional[Dict[str, float]] = None
    ) -> Dict[str, Any]:
        """Update LLM routing configuration"""
        params = {
            "model_name": model_name,
            "strategy": strategy,
        }
        if weights:
            params["weights"] = weights
        return await self._call_mcp("update_routing", params)
    
    async def list_llm_endpoints(
        self,
        filters: Optional[Dict[str, Any]] = None
    ) -> List[Dict[str, Any]]:
        """List available LLM endpoints"""
        params = {"filters": filters or {}}
        result = await self._call_mcp("list_llm_endpoints", params)
        return result.get("endpoints", [])
    
    # ========== Training Tools (5) ==========
    
    async def start_training(
        self,
        base_model: str,
        dataset_path: str,
        training_type: str = "lora",
        **hyperparameters
    ) -> Dict[str, Any]:
        """Start model training job"""
        params = {
            "base_model": base_model,
            "dataset_path": dataset_path,
            "training_type": training_type,
            "hyperparameters": hyperparameters
        }
        return await self._call_mcp("start_training", params)
    
    async def recommend_small_model(
        self,
        dataset_size: int,
        task_type: str,
        budget: Optional[float] = None
    ) -> Dict[str, Any]:
        """Get model recommendation based on requirements"""
        params = {
            "dataset_size": dataset_size,
            "task_type": task_type,
        }
        if budget:
            params["budget"] = budget
        return await self._call_mcp("recommend_small_model", params)
    
    async def list_small_models(self) -> List[Dict[str, Any]]:
        """List available small models for training"""
        result = await self._call_mcp("list_small_models", {})
        return result.get("models", [])
    
    async def train_small_model(
        self,
        model_id: str,
        dataset_path: str,
        **kwargs
    ) -> Dict[str, Any]:
        """Train a specific small model"""
        params = {
            "model_id": model_id,
            "dataset_path": dataset_path,
            **kwargs
        }
        return await self._call_mcp("train_small_model", params)
    
    async def get_training_status(
        self,
        job_id: str
    ) -> Dict[str, Any]:
        """Check training job status"""
        params = {"job_id": job_id}
        return await self._call_mcp("get_training_status", params)
    
    # ========== Monitoring Tools (4) ==========
    
    async def get_metrics(
        self,
        metric_names: List[str],
        time_range: str = "1h",
        filters: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Query platform metrics"""
        params = {
            "metric_names": metric_names,
            "time_range": time_range,
            "filters": filters or {}
        }
        return await self._call_mcp("get_metrics", params)
    
    async def detect_anomalies(
        self,
        metric_name: str,
        threshold: Optional[float] = None
    ) -> Dict[str, Any]:
        """Detect anomalies in metrics"""
        params = {
            "metric_name": metric_name,
        }
        if threshold:
            params["threshold"] = threshold
        return await self._call_mcp("detect_anomalies", params)
    
    async def query_usage(
        self,
        filters: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Query resource usage"""
        params = {"filters": filters}
        return await self._call_mcp("query_usage", params)
    
    async def get_endpoint_health(
        self,
        endpoint_url: str
    ) -> Dict[str, Any]:
        """Check endpoint health status"""
        params = {"endpoint_url": endpoint_url}
        return await self._call_mcp("get_endpoint_health", params)
    
    # ========== Cost Tools (3) ==========
    
    async def get_costs(
        self,
        time_range: str = "7d",
        group_by: Optional[List[str]] = None
    ) -> Dict[str, Any]:
        """Get cost breakdown"""
        params = {
            "time_range": time_range,
            "group_by": group_by or ["tenant", "workload"]
        }
        return await self._call_mcp("get_costs", params)
    
    async def forecast_costs(
        self,
        horizon: str = "30d",
        confidence: float = 0.95
    ) -> Dict[str, Any]:
        """Forecast future costs"""
        params = {
            "horizon": horizon,
            "confidence": confidence
        }
        return await self._call_mcp("forecast_costs", params)
    
    async def recommend_optimization(
        self,
        tenant_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get cost optimization recommendations"""
        params = {}
        if tenant_id:
            params["tenant_id"] = tenant_id
        return await self._call_mcp("recommend_optimization", params)
    
    # ========== Memory Tools (3) ==========
    
    async def provision_agent_memory(
        self,
        agent_id: str,
        memory_types: List[str],
        retention_policy: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Provision memory systems for an agent"""
        params = {
            "agent_id": agent_id,
            "memory_types": memory_types,
            "retention_policy": retention_policy or {}
        }
        return await self._call_mcp("provision_agent_memory", params)
    
    async def store_semantic_memory(
        self,
        agent_id: str,
        content: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """Store semantic knowledge"""
        params = {
            "agent_id": agent_id,
            "content": content,
            "metadata": metadata or {}
        }
        return await self._call_mcp("store_semantic_memory", params)
    
    async def search_memory(
        self,
        agent_id: str,
        query: str,
        memory_types: Optional[List[str]] = None,
        limit: int = 10
    ) -> List[Dict[str, Any]]:
        """Search across memory types"""
        params = {
            "agent_id": agent_id,
            "query": query,
            "memory_types": memory_types or ["semantic", "episodic"],
            "limit": limit
        }
        result = await self._call_mcp("search_memory", params)
        return result.get("results", [])
    
    # ========== Discovery Tool (1) ==========
    
    async def update_endpoint_priority(
        self,
        endpoint_url: str,
        priority: int
    ) -> Dict[str, Any]:
        """Update LLM endpoint routing priority"""
        params = {
            "endpoint_url": endpoint_url,
            "priority": priority
        }
        return await self._call_mcp("update_endpoint_priority", params)
    
    async def close(self):
        """Close HTTP client"""
        await self.client.aclose()

