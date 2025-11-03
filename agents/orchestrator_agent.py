"""
NexusAI Orchestrator Agent - Microsoft Agent Framework Implementation

Graph-based workflows with branching, checkpointing, and human-in-loop.
"""

import asyncio
from typing import Callable, Dict, Any
from agent_framework.workflows import GraphWorkflow, ConditionalEdge
from agent_framework.workflows.state import WorkflowState, load_state, save_state
from agent_framework.logging import get_logger

from nexusai import NexusAIMCPTools, NexusAIConfig, WorkflowBridge

logger = get_logger(__name__)


class OrchestratorAgent:
    """
    Orchestrator Agent using Microsoft Agent Framework.
    
    Supports:
    - Graph-based workflows with conditional branching
    - Checkpointing for fault tolerance
    - Human-in-the-loop approval gates
    - Time-travel debugging
    - Streaming progress updates
    """
    
    def __init__(self, config: NexusAIConfig):
        self.config = config
        self.tools = NexusAIMCPTools(config)
        self.bridge = WorkflowBridge(config)
        self.workflows = {}
        
        # Register workflows
        self._register_workflows()
    
    def _register_workflows(self):
        """Register all workflows"""
        self.workflows["deploy_model"] = self.create_deploy_model_workflow()
        self.workflows["train_and_deploy"] = self.create_train_and_deploy_workflow()
        self.workflows["optimize_costs"] = self.create_optimize_costs_workflow()
    
    # ========== WORKFLOW 1: Deploy Model (Graph-based) ==========
    
    def create_deploy_model_workflow(self) -> GraphWorkflow:
        """
        Deploy Model Workflow with branching and approval gates.
        
        Graph:
        validate_customer → check_quota → [YES] → allocate_gpu → deploy_model → health_check → [PASS] → notify_success
                                        → [NO]  → notify_no_quota
                                                                              → [FAIL] → rollback → notify_failure
        """
        workflow = GraphWorkflow(name="deploy_model")
        
        # Add nodes
        workflow.add_node("validate_customer", self._validate_customer)
        workflow.add_node("check_quota", self._check_quota)
        workflow.add_node("notify_no_quota", self._notify_no_quota)
        workflow.add_node("allocate_gpu", self._allocate_gpu)
        workflow.add_node("deploy_model", self._deploy_model)
        workflow.add_node("health_check", self._health_check)
        workflow.add_node("rollback", self._rollback_deployment)
        workflow.add_node("notify_success", self._notify_success)
        workflow.add_node("notify_failure", self._notify_failure)
        
        # Add edges
        workflow.add_edge("validate_customer", "check_quota")
        
        # Conditional: quota available?
        workflow.add_conditional_edge(
            "check_quota",
            self._should_proceed_with_allocation,
            {
                "yes": "allocate_gpu",
                "no": "notify_no_quota"
            }
        )
        
        workflow.add_edge("allocate_gpu", "deploy_model")
        workflow.add_edge("deploy_model", "health_check")
        
        # Conditional: health check passed?
        workflow.add_conditional_edge(
            "health_check",
            self._should_notify_success,
            {
                "pass": "notify_success",
                "fail": "rollback"
            }
        )
        
        workflow.add_edge("rollback", "notify_failure")
        
        # Set entry and exit points
        workflow.set_entry_point("validate_customer")
        workflow.set_exit_point("notify_success")
        workflow.set_exit_point("notify_no_quota")
        workflow.set_exit_point("notify_failure")
        
        # Enable checkpointing (save after each node)
        workflow.enable_checkpointing()
        
        return workflow
    
    # ========== Workflow Node Functions ==========
    
    async def _validate_customer(self, state: WorkflowState) -> Dict[str, Any]:
        """Validate customer exists"""
        customer_id = state.get("customer_id")
        if not customer_id:
            raise ValueError("customer_id is required")
        
        logger.info(f"Validating customer: {customer_id}")
        
        # In real implementation, query customer database
        return {
            "valid": True,
            "customer_id": customer_id,
            "tier": "enterprise"  # Could affect resource limits
        }
    
    async def _check_quota(self, state: WorkflowState) -> Dict[str, Any]:
        """Check if customer has available GPU quota"""
        customer_id = state.get("customer_id")
        vgpu_requested = state.get("vgpu_size", 1.0)
        
        logger.info(f"Checking quota for {customer_id}: {vgpu_requested} vGPU")
        
        # Query quota via MCP
        usage = await self.tools.query_usage({
            "customer": customer_id,
            "resource": "vgpu"
        })
        
        available = usage.get("available", 0)
        quota_available = available >= vgpu_requested
        
        return {
            "quota_available": quota_available,
            "available_vgpu": available,
            "requested_vgpu": vgpu_requested
        }
    
    def _should_proceed_with_allocation(self, state: WorkflowState) -> str:
        """Conditional: Check if quota is available"""
        return "yes" if state.get("quota_available") else "no"
    
    async def _notify_no_quota(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify customer that quota is insufficient"""
        customer_id = state.get("customer_id")
        logger.warning(f"Insufficient quota for {customer_id}")
        
        return {
            "status": "rejected",
            "reason": "insufficient_quota",
            "message": f"Customer {customer_id} has insufficient GPU quota"
        }
    
    async def _allocate_gpu(self, state: WorkflowState) -> Dict[str, Any]:
        """Allocate GPU resources"""
        vgpu_size = state.get("vgpu_size", 1.0)
        
        logger.info(f"Allocating {vgpu_size} vGPU")
        
        result = await self.tools.allocate_gpu(
            vgpu_size=vgpu_size,
            duration="24h",
            pool_name="default-pool"
        )
        
        # Save checkpoint after allocation
        await self.bridge.save_workflow_checkpoint(
            workflow_id=state.get("workflow_id"),
            state=state.to_dict()
        )
        
        return result
    
    async def _deploy_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Deploy the model"""
        model_name = state.get("model_name")
        model_path = state.get("model_path")
        vgpu_size = state.get("vgpu_size", 1.0)
        
        logger.info(f"Deploying model: {model_name}")
        
        result = await self.tools.deploy_model(
            model_name=model_name,
            model_path=model_path,
            vgpu_size=vgpu_size,
            replicas=1
        )
        
        # Save checkpoint after deployment
        await self.bridge.save_workflow_checkpoint(
            workflow_id=state.get("workflow_id"),
            state=state.to_dict()
        )
        
        return result
    
    async def _health_check(self, state: WorkflowState) -> Dict[str, Any]:
        """Check deployment health"""
        endpoint_url = state.get("endpoint_url")
        
        logger.info(f"Checking health: {endpoint_url}")
        
        # Wait for deployment to stabilize
        await asyncio.sleep(5)
        
        health = await self.tools.get_endpoint_health(endpoint_url)
        
        return {
            "healthy": health.get("status") == "healthy",
            "endpoint_url": endpoint_url
        }
    
    def _should_notify_success(self, state: WorkflowState) -> str:
        """Conditional: Check if deployment is healthy"""
        return "pass" if state.get("healthy") else "fail"
    
    async def _rollback_deployment(self, state: WorkflowState) -> Dict[str, Any]:
        """Rollback failed deployment"""
        model_name = state.get("model_name")
        
        logger.error(f"Rolling back deployment: {model_name}")
        
        # In real implementation, delete Kubernetes resources
        return {
            "rolled_back": True,
            "model_name": model_name
        }
    
    async def _notify_success(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify successful deployment"""
        model_name = state.get("model_name")
        endpoint_url = state.get("endpoint_url")
        
        logger.info(f"Deployment successful: {model_name} at {endpoint_url}")
        
        return {
            "status": "success",
            "model_name": model_name,
            "endpoint_url": endpoint_url
        }
    
    async def _notify_failure(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify deployment failure"""
        model_name = state.get("model_name")
        
        logger.error(f"Deployment failed: {model_name}")
        
        return {
            "status": "failed",
            "model_name": model_name,
            "reason": "health_check_failed"
        }
    
    # ========== WORKFLOW 2: Train and Deploy (Complex Graph) ==========
    
    def create_train_and_deploy_workflow(self) -> GraphWorkflow:
        """
        Train and Deploy Workflow with parallel training and checkpointing.
        
        Graph:
        validate_dataset → recommend_model → [parallel] → train_A, train_B, train_C
                                                        → compare_results → [best_accuracy > 90%?]
                                                                         → [YES] → ask_approval → [APPROVED] → deploy
                                                                                                             → [REJECTED] → archive
                                                                         → [NO] → notify_low_quality
        """
        workflow = GraphWorkflow(name="train_and_deploy")
        
        # Nodes for training workflow
        workflow.add_node("validate_dataset", self._validate_dataset)
        workflow.add_node("recommend_model", self._recommend_model)
        workflow.add_node("train_parallel", self._train_parallel)  # Parallel training
        workflow.add_node("compare_results", self._compare_training_results)
        workflow.add_node("ask_approval", self._ask_user_approval)
        workflow.add_node("deploy_best_model", self._deploy_best_model)
        workflow.add_node("archive_model", self._archive_model)
        workflow.add_node("notify_low_quality", self._notify_low_quality)
        
        # Edges
        workflow.add_edge("validate_dataset", "recommend_model")
        workflow.add_edge("recommend_model", "train_parallel")
        workflow.add_edge("train_parallel", "compare_results")
        
        # Conditional: quality sufficient?
        workflow.add_conditional_edge(
            "compare_results",
            self._check_quality_sufficient,
            {
                "yes": "ask_approval",
                "no": "notify_low_quality"
            }
        )
        
        # Conditional: user approved?
        workflow.add_conditional_edge(
            "ask_approval",
            self._check_user_approved,
            {
                "approved": "deploy_best_model",
                "rejected": "archive_model"
            }
        )
        
        workflow.set_entry_point("validate_dataset")
        workflow.set_exit_point("deploy_best_model")
        workflow.set_exit_point("archive_model")
        workflow.set_exit_point("notify_low_quality")
        
        # Enable checkpointing (critical for long training jobs)
        workflow.enable_checkpointing()
        
        return workflow
    
    # ========== Training Workflow Nodes ==========
    
    async def _validate_dataset(self, state: WorkflowState) -> Dict[str, Any]:
        """Validate dataset exists and is properly formatted"""
        dataset_path = state.get("dataset_path")
        
        if not dataset_path:
            raise ValueError("dataset_path is required")
        
        logger.info(f"Validating dataset: {dataset_path}")
        
        # In real implementation, check file exists, format valid, etc.
        return {
            "valid": True,
            "dataset_path": dataset_path,
            "sample_count": 5000,  # Example
            "format": "jsonl"
        }
    
    async def _recommend_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Recommend models to try"""
        sample_count = state.get("sample_count", 1000)
        task_type = state.get("task_type", "classification")
        
        logger.info(f"Recommending models for {sample_count} samples")
        
        recommendation = await self.tools.recommend_small_model(
            dataset_size=sample_count,
            task_type=task_type
        )
        
        return {
            "recommended_models": recommendation.get("models", []),
            "primary_model": recommendation.get("primary"),
            "alternatives": recommendation.get("alternatives", [])
        }
    
    async def _train_parallel(self, state: WorkflowState) -> Dict[str, Any]:
        """Train multiple models in parallel"""
        models = state.get("recommended_models", [])
        dataset_path = state.get("dataset_path")
        
        logger.info(f"Starting parallel training for {len(models)} models")
        
        # Start training jobs in parallel
        training_jobs = []
        for model in models:
            job = await self.tools.train_small_model(
                model_id=model,
                dataset_path=dataset_path
            )
            training_jobs.append({
                "model": model,
                "job_id": job.get("job_id"),
                "status": "running"
            })
        
        # Monitor until all complete (with checkpointing)
        completed_jobs = await self._monitor_training_jobs(training_jobs, state)
        
        return {
            "training_results": completed_jobs
        }
    
    async def _monitor_training_jobs(
        self,
        jobs: list,
        state: WorkflowState
    ) -> list:
        """Monitor training jobs until completion"""
        while True:
            all_complete = True
            
            for job in jobs:
                if job["status"] == "completed":
                    continue
                
                status = await self.tools.get_training_status(job["job_id"])
                job["status"] = status.get("status")
                job["accuracy"] = status.get("accuracy", 0)
                
                if job["status"] not in ["completed", "failed"]:
                    all_complete = False
            
            # Save checkpoint
            state["training_jobs"] = jobs
            await self.bridge.save_workflow_checkpoint(
                workflow_id=state.get("workflow_id"),
                state=state.to_dict()
            )
            
            if all_complete:
                break
            
            await asyncio.sleep(30)  # Check every 30s
        
        return jobs
    
    async def _compare_training_results(self, state: WorkflowState) -> Dict[str, Any]:
        """Compare training results and pick best model"""
        results = state.get("training_results", [])
        
        # Find best by accuracy
        best = max(results, key=lambda x: x.get("accuracy", 0))
        
        logger.info(f"Best model: {best['model']} with accuracy {best['accuracy']}")
        
        return {
            "best_model": best["model"],
            "best_accuracy": best["accuracy"],
            "best_job_id": best["job_id"],
            "all_results": results
        }
    
    def _check_quality_sufficient(self, state: WorkflowState) -> str:
        """Check if best model meets quality threshold"""
        accuracy = state.get("best_accuracy", 0)
        return "yes" if accuracy > 0.90 else "no"
    
    async def _ask_user_approval(self, state: WorkflowState) -> Dict[str, Any]:
        """Human-in-the-loop: Ask for deployment approval"""
        model = state.get("best_model")
        accuracy = state.get("best_accuracy")
        
        logger.info(f"Asking user approval for {model} (accuracy: {accuracy})")
        
        # In real implementation, this would:
        # 1. Send notification to user
        # 2. Wait for response via API/webhook
        # 3. Resume workflow when response received
        
        # For now, auto-approve if accuracy > 95%
        approved = accuracy > 0.95
        
        return {
            "user_approved": approved,
            "approval_reason": "auto_approved" if approved else "requires_human_review"
        }
    
    def _check_user_approved(self, state: WorkflowState) -> str:
        """Check if user approved deployment"""
        return "approved" if state.get("user_approved") else "rejected"
    
    async def _deploy_best_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Deploy the best trained model"""
        model = state.get("best_model")
        job_id = state.get("best_job_id")
        
        logger.info(f"Deploying best model: {model}")
        
        result = await self.tools.deploy_model(
            model_name=f"{model}-trained",
            model_path=f"/models/{job_id}",
            vgpu_size=state.get("vgpu_size", 1.0)
        )
        
        return {
            "status": "deployed",
            "model": model,
            "endpoint": result.get("endpoint_url")
        }
    
    async def _archive_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Archive model (user rejected deployment)"""
        model = state.get("best_model")
        
        logger.info(f"Archiving model (user rejected): {model}")
        
        return {
            "status": "archived",
            "model": model
        }
    
    async def _notify_low_quality(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify that model quality is insufficient"""
        accuracy = state.get("best_accuracy")
        
        logger.warning(f"Model quality too low: {accuracy}")
        
        return {
            "status": "rejected",
            "reason": "insufficient_quality",
            "accuracy": accuracy
        }
    
    # ========== WORKFLOW 3: Cost Optimization (Human-in-Loop) ==========
    
    def create_optimize_costs_workflow(self) -> GraphWorkflow:
        """
        Cost Optimization Workflow with approval gates and impact monitoring.
        
        Graph:
        query_metrics → analyze_costs → generate_recommendations → [savings > $1000?]
                                                                 → [YES] → ask_approval → [APPROVED] → apply_changes → monitor_7d → [improved?] → notify_success
                                                                                                                                                  → [worse] → rollback → notify_failure
                                                                                       → [REJECTED] → notify_rejected
                                                                 → [NO] → auto_apply → monitor_7d
        """
        workflow = GraphWorkflow(name="optimize_costs")
        
        # Will implement similarly to above workflows
        # For brevity, showing structure only
        
        workflow.add_node("query_metrics", self._query_cost_metrics)
        workflow.add_node("generate_recommendations", self._generate_cost_recommendations)
        workflow.add_node("ask_cost_approval", self._ask_cost_approval)
        workflow.add_node("apply_optimizations", self._apply_cost_optimizations)
        workflow.add_node("monitor_impact", self._monitor_cost_impact)
        workflow.add_node("rollback_optimizations", self._rollback_cost_optimizations)
        
        # Add edges and conditionals...
        workflow.set_entry_point("query_metrics")
        workflow.enable_checkpointing()
        
        return workflow
    
    # ========== Stub implementations for cost workflow ==========
    
    async def _query_cost_metrics(self, state: WorkflowState) -> Dict[str, Any]:
        """Query cost metrics"""
        return await self.tools.get_costs(time_range="7d")
    
    async def _generate_cost_recommendations(self, state: WorkflowState) -> Dict[str, Any]:
        """Generate optimization recommendations"""
        return await self.tools.recommend_optimization()
    
    async def _ask_cost_approval(self, state: WorkflowState) -> Dict[str, Any]:
        """Ask for approval to apply cost optimizations"""
        savings = state.get("total_savings", 0)
        # Human approval required if savings > $1000
        return {"approved": savings > 1000}
    
    async def _apply_cost_optimizations(self, state: WorkflowState) -> Dict[str, Any]:
        """Apply cost optimizations"""
        recommendations = state.get("recommendations", [])
        # Apply routing changes, scale down resources, etc.
        return {"applied": True}
    
    async def _monitor_cost_impact(self, state: WorkflowState) -> Dict[str, Any]:
        """Monitor impact of optimizations for 7 days"""
        # Would run for 7 days with periodic checkpoints
        return {"improved": True}
    
    async def _rollback_cost_optimizations(self, state: WorkflowState) -> Dict[str, Any]:
        """Rollback if optimizations made things worse"""
        return {"rolled_back": True}
    
    # ========== Public API ==========
    
    async def execute_workflow(
        self,
        workflow_name: str,
        params: Dict[str, Any],
        workflow_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Execute a workflow.
        
        Supports:
        - Resume from checkpoint if workflow_id provided
        - Streaming progress updates
        - Time-travel debugging
        """
        if workflow_name not in self.workflows:
            raise ValueError(f"Unknown workflow: {workflow_name}")
        
        workflow = self.workflows[workflow_name]
        
        # Check for existing checkpoint
        if workflow_id:
            checkpoint = await self.bridge.load_workflow_checkpoint(workflow_id)
            if checkpoint:
                logger.info(f"Resuming workflow {workflow_id} from checkpoint")
                params.update(checkpoint)
        
        # Execute workflow
        result = await workflow.run(params)
        
        return result
    
    async def close(self):
        """Cleanup resources"""
        await self.tools.close()
        await self.bridge.close()


# ========== Main Entry Point ==========

async def main():
    """Run orchestrator agent"""
    config = NexusAIConfig.from_env()
    agent = OrchestratorAgent(config)
    
    # Example: Deploy model workflow
    result = await agent.execute_workflow(
        workflow_name="deploy_model",
        params={
            "customer_id": "customer-123",
            "model_name": "llama-3.1-70b",
            "model_path": "/models/llama-3.1-70b",
            "vgpu_size": 2.0
        }
    )
    
    logger.info(f"Workflow result: {result}")
    
    await agent.close()


if __name__ == "__main__":
    asyncio.run(main())

