"""
NexusAI SmallModel Agent - Microsoft Agent Framework Implementation

Interactive agent for small model selection, training, and deployment.
"""

import asyncio
from typing import Dict, Any, List
from agent_framework.workflows import GraphWorkflow
from agent_framework.workflows.state import WorkflowState
from agent_framework.logging import get_logger

from nexusai import NexusAIMCPTools, NexusAIConfig, WorkflowBridge

logger = get_logger(__name__)


class SmallModelAgent:
    """
    SmallModel Agent with:
    - Interactive model recommendation
    - Dataset analysis
    - Training orchestration (delegates to Training Agent)
    - Model comparison
    """
    
    def __init__(self, config: NexusAIConfig):
        self.config = config
        self.tools = NexusAIMCPTools(config)
        self.bridge = WorkflowBridge(config)
        self.workflows = {}
        
        self._register_workflows()
    
    def _register_workflows(self):
        """Register workflows"""
        self.workflows["recommend_and_train"] = self.create_recommendation_workflow()
    
    def create_recommendation_workflow(self) -> GraphWorkflow:
        """
        Model Recommendation and Training Workflow
        
        Graph:
        analyze_dataset → list_models → recommend_best → estimate_costs → ask_user_confirmation → [APPROVED] → train_model → deploy
                                                                                                → [REJECTED] → ask_for_preferences → recommend_best
        """
        workflow = GraphWorkflow(name="recommend_and_train")
        
        workflow.add_node("analyze_dataset", self._analyze_dataset)
        workflow.add_node("list_models", self._list_available_models)
        workflow.add_node("recommend_best", self._recommend_best_model)
        workflow.add_node("estimate_costs", self._estimate_training_costs)
        workflow.add_node("ask_confirmation", self._ask_user_confirmation)
        workflow.add_node("ask_preferences", self._ask_for_preferences)
        workflow.add_node("train_model", self._train_recommended_model)
        workflow.add_node("deploy_model", self._deploy_trained_model)
        workflow.add_node("notify_cancelled", self._notify_cancelled)
        
        # Edges
        workflow.add_edge("analyze_dataset", "list_models")
        workflow.add_edge("list_models", "recommend_best")
        workflow.add_edge("recommend_best", "estimate_costs")
        workflow.add_edge("estimate_costs", "ask_confirmation")
        
        # Conditional: user confirmed?
        workflow.add_conditional_edge(
            "ask_confirmation",
            lambda s: "approved" if s.get("user_confirmed") else "rejected",
            {
                "approved": "train_model",
                "rejected": "ask_preferences"
            }
        )
        
        # If rejected, get preferences and re-recommend
        workflow.add_edge("ask_preferences", "recommend_best")
        
        workflow.add_edge("train_model", "deploy_model")
        
        workflow.set_entry_point("analyze_dataset")
        workflow.set_exit_point("deploy_model")
        workflow.set_exit_point("notify_cancelled")
        
        workflow.enable_checkpointing()
        
        return workflow
    
    # Node implementations
    
    async def _analyze_dataset(self, state: WorkflowState) -> Dict[str, Any]:
        """Analyze dataset characteristics"""
        dataset_path = state.get("dataset_path")
        
        if not dataset_path:
            raise ValueError("dataset_path is required")
        
        logger.info(f"Analyzing dataset: {dataset_path}")
        
        # In real implementation:
        # - Read dataset file
        # - Count samples
        # - Detect language
        # - Identify domain (code, chat, etc.)
        # - Detect format (JSONL, CSV, Parquet)
        
        return {
            "sample_count": 5000,
            "language": "en",
            "domain": "chat",
            "format": "jsonl",
            "avg_token_length": 256
        }
    
    async def _list_available_models(self, state: WorkflowState) -> Dict[str, Any]:
        """List available small models"""
        logger.info("Listing available small models")
        
        models = await self.tools.list_small_models()
        
        return {
            "available_models": models
        }
    
    async def _recommend_best_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Recommend best model based on dataset"""
        sample_count = state.get("sample_count", 1000)
        domain = state.get("domain", "general")
        task_type = state.get("task_type", "chat")
        budget = state.get("budget_limit")
        user_preferences = state.get("user_preferences", {})
        
        logger.info(f"Recommending model for {sample_count} samples ({domain}/{task_type})")
        
        # Get recommendation from Model Catalog
        recommendation = await self.tools.recommend_small_model(
            dataset_size=sample_count,
            task_type=task_type,
            budget=budget
        )
        
        # Apply user preferences if provided
        preferred_size = user_preferences.get("preferred_size")
        if preferred_size:
            # Filter by size
            pass
        
        return {
            "recommended_model": recommendation.get("primary"),
            "alternatives": recommendation.get("alternatives", []),
            "reasoning": recommendation.get("reasoning")
        }
    
    async def _estimate_training_costs(self, state: WorkflowState) -> Dict[str, Any]:
        """Estimate training costs"""
        model = state.get("recommended_model", {})
        sample_count = state.get("sample_count", 1000)
        
        model_size = model.get("parameters", "7b")
        
        # Rough cost estimation
        if "70b" in model_size.lower():
            vgpu_required = 4.0
            hours_estimate = 6
        elif "13b" in model_size.lower():
            vgpu_required = 2.0
            hours_estimate = 4
        else:
            vgpu_required = 1.0
            hours_estimate = 2
        
        # Cost calculation (simplified: $1/vGPU/hour)
        estimated_cost = vgpu_required * hours_estimate
        
        logger.info(f"Estimated cost: ${estimated_cost} ({vgpu_required} vGPU × {hours_estimate}h)")
        
        return {
            "vgpu_required": vgpu_required,
            "estimated_hours": hours_estimate,
            "estimated_cost": estimated_cost,
            "cost_breakdown": {
                "gpu": estimated_cost * 0.7,
                "storage": estimated_cost * 0.2,
                "network": estimated_cost * 0.1
            }
        }
    
    async def _ask_user_confirmation(self, state: WorkflowState) -> Dict[str, Any]:
        """Ask user to confirm model selection"""
        model = state.get("recommended_model", {})
        cost = state.get("estimated_cost", 0)
        hours = state.get("estimated_hours", 0)
        
        logger.info(
            f"Asking user confirmation: {model.get('name')} - "
            f"${cost} (~{hours}h training)"
        )
        
        # In real implementation:
        # - Send notification with details
        # - Present alternatives
        # - Wait for user response
        
        # For demo, auto-confirm if cost < $20
        confirmed = cost < 20
        
        return {
            "user_confirmed": confirmed,
            "confirmation_reason": "auto_confirmed" if confirmed else "requires_human_approval"
        }
    
    async def _ask_for_preferences(self, state: WorkflowState) -> Dict[str, Any]:
        """Ask user for preferences to refine recommendation"""
        logger.info("Asking for user preferences")
        
        # In real implementation:
        # - Ask: prefer speed or accuracy?
        # - Ask: model size preference?
        # - Ask: budget limit?
        
        # For demo, set some preferences
        return {
            "user_preferences": {
                "preferred_size": "small",
                "priority": "speed",
                "max_cost": 10
            }
        }
    
    async def _train_recommended_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Train the recommended model (delegates to Training Agent)"""
        model = state.get("recommended_model", {})
        dataset_path = state.get("dataset_path")
        model_id = model.get("id", "tinyllama")
        
        logger.info(f"Starting training: {model_id}")
        
        # Delegate to Training Agent via MCP
        result = await self.tools.train_small_model(
            model_id=model_id,
            dataset_path=dataset_path,
            num_epochs=3,
            learning_rate=2e-4
        )
        
        job_id = result.get("job_id")
        
        # Monitor training (simplified)
        while True:
            await asyncio.sleep(30)
            
            status = await self.tools.get_training_status(job_id)
            
            if status.get("status") == "completed":
                break
            elif status.get("status") in ["failed", "error"]:
                raise Exception(f"Training failed: {status.get('error')}")
            
            # Checkpoint
            state["training_progress"] = status.get("progress", 0)
            await self.bridge.save_workflow_checkpoint(
                workflow_id=state.get("workflow_id"),
                state=state.to_dict()
            )
        
        return {
            "trained_model_path": result.get("model_path"),
            "training_accuracy": status.get("accuracy", 0),
            "job_id": job_id
        }
    
    async def _deploy_trained_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Deploy the trained model"""
        model_path = state.get("trained_model_path")
        model = state.get("recommended_model", {})
        
        model_name = f"{model.get('name', 'custom')}-trained"
        
        logger.info(f"Deploying trained model: {model_name}")
        
        result = await self.tools.deploy_model(
            model_name=model_name,
            model_path=model_path,
            vgpu_size=1.0,
            replicas=1
        )
        
        return {
            "status": "deployed",
            "model_name": model_name,
            "endpoint_url": result.get("endpoint_url")
        }
    
    async def _notify_cancelled(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify that workflow was cancelled"""
        logger.warning("Workflow cancelled by user")
        
        return {"status": "cancelled"}
    
    async def execute(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute workflow"""
        workflow = self.workflows["recommend_and_train"]
        result = await workflow.run(params)
        
        return result
    
    async def close(self):
        """Cleanup"""
        await self.tools.close()
        await self.bridge.close()


async def main():
    """Run SmallModel agent"""
    config = NexusAIConfig.from_env()
    agent = SmallModelAgent(config)
    
    result = await agent.execute({
        "dataset_path": "/data/my-chat-dataset.jsonl",
        "task_type": "chat",
        "budget_limit": 50
    })
    
    logger.info(f"Result: {result}")
    
    await agent.close()


if __name__ == "__main__":
    asyncio.run(main())

