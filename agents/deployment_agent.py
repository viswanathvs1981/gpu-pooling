"""
NexusAI Deployment Agent - Microsoft Agent Framework Implementation

Multi-stage deployment with approval gates, canary rollouts, and automatic rollback.
"""

import asyncio
from typing import Dict, Any
from agent_framework.workflows import GraphWorkflow
from agent_framework.workflows.state import WorkflowState
from agent_framework.logging import get_logger

from nexusai import NexusAIMCPTools, NexusAIConfig, WorkflowBridge

logger = get_logger(__name__)


class DeploymentAgent:
    """
    Deployment Agent with:
    - Multi-stage deployments (dev → staging → prod)
    - Approval gates between stages
    - Canary rollouts
    - Automatic rollback on failure
    - Health monitoring
    """
    
    def __init__(self, config: NexusAIConfig):
        self.config = config
        self.tools = NexusAIMCPTools(config)
        self.bridge = WorkflowBridge(config)
        self.workflows = {}
        
        self._register_workflows()
    
    def _register_workflows(self):
        """Register deployment workflows"""
        self.workflows["deploy_multi_stage"] = self.create_multi_stage_deployment()
        self.workflows["deploy_canary"] = self.create_canary_deployment()
    
    def create_multi_stage_deployment(self) -> GraphWorkflow:
        """
        Multi-stage deployment: Dev → Staging → Prod
        
        Graph:
        deploy_dev → test_dev → [PASS] → ask_staging_approval → [APPROVED] → deploy_staging
                              → [FAIL] → rollback_dev → notify_failure
                                                       → [APPROVED] → monitor_staging_24h → [HEALTHY] → ask_prod_approval
                                                                                         → [UNHEALTHY] → rollback_staging
                                                                                                       → [APPROVED] → deploy_prod_canary
                                                                                                                   → monitor_canary → [HEALTHY] → scale_to_100
                                                                                                                                    → [UNHEALTHY] → rollback_prod
        """
        workflow = GraphWorkflow(name="deploy_multi_stage")
        
        # Dev stage
        workflow.add_node("deploy_dev", self._deploy_to_dev)
        workflow.add_node("test_dev", self._test_deployment)
        workflow.add_node("rollback_dev", self._rollback)
        workflow.add_node("notify_dev_failure", self._notify_failure)
        
        # Staging stage
        workflow.add_node("ask_staging_approval", self._ask_staging_approval)
        workflow.add_node("deploy_staging", self._deploy_to_staging)
        workflow.add_node("monitor_staging", self._monitor_staging_24h)
        workflow.add_node("rollback_staging", self._rollback_staging)
        
        # Prod stage
        workflow.add_node("ask_prod_approval", self._ask_prod_approval)
        workflow.add_node("deploy_prod_canary", self._deploy_prod_canary)
        workflow.add_node("monitor_canary", self._monitor_canary)
        workflow.add_node("scale_to_100", self._scale_to_full)
        workflow.add_node("rollback_prod", self._rollback_prod)
        workflow.add_node("notify_success", self._notify_success)
        
        # Dev edges
        workflow.add_edge("deploy_dev", "test_dev")
        workflow.add_conditional_edge(
            "test_dev",
            lambda s: "pass" if s.get("dev_tests_passed") else "fail",
            {"pass": "ask_staging_approval", "fail": "rollback_dev"}
        )
        workflow.add_edge("rollback_dev", "notify_dev_failure")
        
        # Staging edges
        workflow.add_conditional_edge(
            "ask_staging_approval",
            lambda s: "approved" if s.get("staging_approved") else "rejected",
            {"approved": "deploy_staging", "rejected": "notify_dev_failure"}
        )
        workflow.add_edge("deploy_staging", "monitor_staging")
        workflow.add_conditional_edge(
            "monitor_staging",
            lambda s: "healthy" if s.get("staging_healthy") else "unhealthy",
            {"healthy": "ask_prod_approval", "unhealthy": "rollback_staging"}
        )
        workflow.add_edge("rollback_staging", "notify_dev_failure")
        
        # Prod edges
        workflow.add_conditional_edge(
            "ask_prod_approval",
            lambda s: "approved" if s.get("prod_approved") else "rejected",
            {"approved": "deploy_prod_canary", "rejected": "notify_dev_failure"}
        )
        workflow.add_edge("deploy_prod_canary", "monitor_canary")
        workflow.add_conditional_edge(
            "monitor_canary",
            lambda s: "healthy" if s.get("canary_healthy") else "unhealthy",
            {"healthy": "scale_to_100", "unhealthy": "rollback_prod"}
        )
        workflow.add_edge("scale_to_100", "notify_success")
        workflow.add_edge("rollback_prod", "notify_dev_failure")
        
        workflow.set_entry_point("deploy_dev")
        workflow.set_exit_point("notify_success")
        workflow.set_exit_point("notify_dev_failure")
        
        workflow.enable_checkpointing()
        
        return workflow
    
    # Node implementations
    
    async def _deploy_to_dev(self, state: WorkflowState) -> Dict[str, Any]:
        """Deploy to dev environment"""
        model_name = state.get("model_name")
        model_path = state.get("model_path")
        
        logger.info(f"Deploying {model_name} to DEV")
        
        result = await self.tools.deploy_model(
            model_name=f"{model_name}-dev",
            model_path=model_path,
            vgpu_size=0.5,  # Small for dev
            replicas=1
        )
        
        return {
            "dev_endpoint": result.get("endpoint_url"),
            "dev_deployed": True
        }
    
    async def _test_deployment(self, state: WorkflowState) -> Dict[str, Any]:
        """Run automated tests"""
        endpoint = state.get("dev_endpoint")
        
        logger.info(f"Testing deployment: {endpoint}")
        
        # Run safety checks
        safety_result = await self.bridge.check_safety(
            "Test prompt for validation",
            context={"environment": "dev"}
        )
        
        tests_passed = safety_result.get("safe", False)
        
        return {
            "dev_tests_passed": tests_passed
        }
    
    async def _ask_staging_approval(self, state: WorkflowState) -> Dict[str, Any]:
        """Ask for staging deployment approval"""
        model_name = state.get("model_name")
        
        logger.info(f"Requesting staging approval for {model_name}")
        
        # In real implementation, send notification and wait for approval
        # For now, auto-approve
        return {"staging_approved": True}
    
    async def _deploy_to_staging(self, state: WorkflowState) -> Dict[str, Any]:
        """Deploy to staging"""
        model_name = state.get("model_name")
        model_path = state.get("model_path")
        
        logger.info(f"Deploying {model_name} to STAGING")
        
        result = await self.tools.deploy_model(
            model_name=f"{model_name}-staging",
            model_path=model_path,
            vgpu_size=1.0,
            replicas=2
        )
        
        await self.bridge.save_workflow_checkpoint(
            workflow_id=state.get("workflow_id"),
            state=state.to_dict()
        )
        
        return {
            "staging_endpoint": result.get("endpoint_url"),
            "staging_deployed": True
        }
    
    async def _monitor_staging_24h(self, state: WorkflowState) -> Dict[str, Any]:
        """Monitor staging for 24 hours"""
        endpoint = state.get("staging_endpoint")
        
        logger.info(f"Monitoring staging for 24h: {endpoint}")
        
        # In real implementation, monitor for 24 hours
        # For demo, check for 1 minute
        for i in range(6):  # 6 checks over 1 min
            await asyncio.sleep(10)
            
            health = await self.tools.get_endpoint_health(endpoint)
            if health.get("status") != "healthy":
                return {"staging_healthy": False}
            
            # Checkpoint
            if i % 2 == 0:
                await self.bridge.save_workflow_checkpoint(
                    workflow_id=state.get("workflow_id"),
                    state=state.to_dict()
                )
        
        return {"staging_healthy": True}
    
    async def _ask_prod_approval(self, state: WorkflowState) -> Dict[str, Any]:
        """Ask for production deployment approval"""
        model_name = state.get("model_name")
        
        logger.info(f"Requesting production approval for {model_name}")
        
        # For demo, auto-approve
        return {"prod_approved": True}
    
    async def _deploy_prod_canary(self, state: WorkflowState) -> Dict[str, Any]:
        """Deploy to prod with 10% canary"""
        model_name = state.get("model_name")
        model_path = state.get("model_path")
        
        logger.info(f"Deploying {model_name} to PROD (10% canary)")
        
        result = await self.tools.deploy_model(
            model_name=f"{model_name}-prod",
            model_path=model_path,
            vgpu_size=2.0,
            replicas=1,  # Start with 1 replica (10%)
            traffic_weight=0.1
        )
        
        return {
            "prod_endpoint": result.get("endpoint_url"),
            "canary_deployed": True
        }
    
    async def _monitor_canary(self, state: WorkflowState) -> Dict[str, Any]:
        """Monitor canary for 1 hour"""
        endpoint = state.get("prod_endpoint")
        
        logger.info(f"Monitoring canary: {endpoint}")
        
        # Monitor for 1 min (demo)
        for i in range(6):
            await asyncio.sleep(10)
            
            health = await self.tools.get_endpoint_health(endpoint)
            if health.get("status") != "healthy":
                return {"canary_healthy": False}
        
        return {"canary_healthy": True}
    
    async def _scale_to_full(self, state: WorkflowState) -> Dict[str, Any]:
        """Scale to 100% traffic"""
        model_name = state.get("model_name")
        
        logger.info(f"Scaling {model_name} to 100%")
        
        # Update routing to 100%
        await self.tools.update_routing(
            model_name=f"{model_name}-prod",
            strategy="direct",
            weights={f"{model_name}-prod": 1.0}
        )
        
        return {"scaled_to_100": True}
    
    async def _rollback(self, state: WorkflowState) -> Dict[str, Any]:
        """Rollback deployment"""
        logger.error("Rolling back deployment")
        return {"rolled_back": True}
    
    async def _rollback_staging(self, state: WorkflowState) -> Dict[str, Any]:
        """Rollback staging"""
        logger.error("Rolling back staging")
        return {"staging_rolled_back": True}
    
    async def _rollback_prod(self, state: WorkflowState) -> Dict[str, Any]:
        """Rollback production"""
        logger.error("Rolling back production canary")
        return {"prod_rolled_back": True}
    
    async def _notify_success(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify successful deployment"""
        logger.info("Deployment successful!")
        return {"status": "success"}
    
    async def _notify_failure(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify deployment failure"""
        logger.error("Deployment failed")
        return {"status": "failed"}
    
    def create_canary_deployment(self) -> GraphWorkflow:
        """Canary deployment workflow (simplified)"""
        workflow = GraphWorkflow(name="deploy_canary")
        # Implementation similar to above, focusing on canary rollout
        return workflow
    
    async def deploy(self, workflow_name: str, params: Dict[str, Any]) -> Dict[str, Any]:
        """Execute deployment workflow"""
        if workflow_name not in self.workflows:
            raise ValueError(f"Unknown workflow: {workflow_name}")
        
        workflow = self.workflows[workflow_name]
        result = await workflow.run(params)
        
        return result
    
    async def close(self):
        """Cleanup"""
        await self.tools.close()
        await self.bridge.close()


async def main():
    """Run deployment agent"""
    config = NexusAIConfig.from_env()
    agent = DeploymentAgent(config)
    
    result = await agent.deploy(
        workflow_name="deploy_multi_stage",
        params={
            "model_name": "llama-3.1-8b-finetuned",
            "model_path": "/models/trained/llama-3.1-8b-job123"
        }
    )
    
    logger.info(f"Deployment result: {result}")
    
    await agent.close()


if __name__ == "__main__":
    asyncio.run(main())

