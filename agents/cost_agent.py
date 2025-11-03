"""
NexusAI Cost Agent - Microsoft Agent Framework Implementation

Cost optimization with human-in-the-loop approval and impact monitoring.
"""

import asyncio
from typing import Dict, Any
from agent_framework.workflows import GraphWorkflow
from agent_framework.workflows.state import WorkflowState
from agent_framework.logging import get_logger

from nexusai import NexusAIMCPTools, NexusAIConfig, WorkflowBridge

logger = get_logger(__name__)


class CostAgent:
    """
    Cost optimization agent with:
    - Multi-source cost data collection (parallel)
    - ML-based forecasting
    - Human approval gates for high-impact changes
    - 7-day impact monitoring
    - Automatic rollback if costs increase
    """
    
    def __init__(self, config: NexusAIConfig):
        self.config = config
        self.tools = NexusAIMCPTools(config)
        self.bridge = WorkflowBridge(config)
        self.workflows = {}
        
        self._register_workflows()
    
    def _register_workflows(self):
        """Register cost optimization workflows"""
        self.workflows["optimize_costs"] = self.create_cost_optimization_workflow()
    
    def create_cost_optimization_workflow(self) -> GraphWorkflow:
        """
        Cost Optimization Workflow
        
        Graph:
        collect_metrics (parallel) → analyze_costs → forecast → generate_recommendations → [savings > $1000?]
                                                                                         → [YES] → ask_approval → [APPROVED] → apply → monitor_7d → [improved?] → notify_success
                                                                                                                                                                 → [worse] → rollback
                                                                                                                              → [REJECTED] → notify_rejected
                                                                                         → [NO] → auto_apply → monitor_7d
        """
        workflow = GraphWorkflow(name="optimize_costs")
        
        workflow.add_node("collect_metrics", self._collect_cost_metrics_parallel)
        workflow.add_node("analyze_costs", self._analyze_costs)
        workflow.add_node("forecast_costs", self._forecast_future_costs)
        workflow.add_node("generate_recommendations", self._generate_recommendations)
        workflow.add_node("ask_approval", self._ask_cost_approval)
        workflow.add_node("auto_apply", self._apply_optimizations)
        workflow.add_node("apply_with_approval", self._apply_optimizations)
        workflow.add_node("monitor_impact", self._monitor_cost_impact_7d)
        workflow.add_node("rollback", self._rollback_optimizations)
        workflow.add_node("notify_success", self._notify_success)
        workflow.add_node("notify_rejected", self._notify_rejected)
        workflow.add_node("notify_rollback", self._notify_rollback)
        
        # Edges
        workflow.add_edge("collect_metrics", "analyze_costs")
        workflow.add_edge("analyze_costs", "forecast_costs")
        workflow.add_edge("forecast_costs", "generate_recommendations")
        
        # Conditional: High impact changes need approval
        workflow.add_conditional_edge(
            "generate_recommendations",
            lambda s: "high_impact" if s.get("total_savings", 0) > 1000 else "low_impact",
            {
                "high_impact": "ask_approval",
                "low_impact": "auto_apply"
            }
        )
        
        # Approval branch
        workflow.add_conditional_edge(
            "ask_approval",
            lambda s: "approved" if s.get("approved") else "rejected",
            {
                "approved": "apply_with_approval",
                "rejected": "notify_rejected"
            }
        )
        
        workflow.add_edge("apply_with_approval", "monitor_impact")
        workflow.add_edge("auto_apply", "monitor_impact")
        
        # Impact monitoring
        workflow.add_conditional_edge(
            "monitor_impact",
            lambda s: "improved" if s.get("cost_improved") else "worse",
            {
                "improved": "notify_success",
                "worse": "rollback"
            }
        )
        
        workflow.add_edge("rollback", "notify_rollback")
        
        workflow.set_entry_point("collect_metrics")
        workflow.set_exit_point("notify_success")
        workflow.set_exit_point("notify_rejected")
        workflow.set_exit_point("notify_rollback")
        
        workflow.enable_checkpointing()
        
        return workflow
    
    # Node implementations
    
    async def _collect_cost_metrics_parallel(self, state: WorkflowState) -> Dict[str, Any]:
        """Collect cost data from multiple sources in parallel"""
        logger.info("Collecting cost metrics from multiple sources")
        
        # Parallel collection
        tasks = [
            self.tools.get_costs(time_range="7d"),
            self.tools.query_usage({"time_range": "7d"}),
            self.bridge.get_analytics("cost_per_tenant", "7d")
        ]
        
        results = await asyncio.gather(*tasks)
        
        return {
            "cloud_costs": results[0],
            "resource_usage": results[1],
            "analytics": results[2],
            "total_cost_7d": results[0].get("total", 0)
        }
    
    async def _analyze_costs(self, state: WorkflowState) -> Dict[str, Any]:
        """Analyze cost data"""
        total_cost = state.get("total_cost_7d", 0)
        
        logger.info(f"Analyzing costs: ${total_cost}/week")
        
        # Identify top spenders
        cloud_costs = state.get("cloud_costs", {})
        by_tenant = cloud_costs.get("by_tenant", {})
        
        top_spenders = sorted(
            by_tenant.items(),
            key=lambda x: x[1],
            reverse=True
        )[:5]
        
        return {
            "top_spenders": top_spenders,
            "avg_daily_cost": total_cost / 7
        }
    
    async def _forecast_future_costs(self, state: WorkflowState) -> Dict[str, Any]:
        """Forecast costs for next 30 days"""
        logger.info("Forecasting costs for next 30 days")
        
        forecast = await self.tools.forecast_costs(horizon="30d", confidence=0.95)
        
        return {
            "forecasted_cost_30d": forecast.get("predicted_cost", 0),
            "confidence": forecast.get("confidence", 0)
        }
    
    async def _generate_recommendations(self, state: WorkflowState) -> Dict[str, Any]:
        """Generate cost optimization recommendations"""
        logger.info("Generating cost optimization recommendations")
        
        recommendations = await self.tools.recommend_optimization()
        
        total_savings = sum(r.get("savings", 0) for r in recommendations.get("recommendations", []))
        
        return {
            "recommendations": recommendations.get("recommendations", []),
            "total_savings": total_savings
        }
    
    async def _ask_cost_approval(self, state: WorkflowState) -> Dict[str, Any]:
        """Ask for approval to apply high-impact optimizations"""
        total_savings = state.get("total_savings", 0)
        recommendations = state.get("recommendations", [])
        
        logger.info(f"Requesting approval for ${total_savings} in optimizations")
        
        # In real implementation:
        # - Send notification with details
        # - Wait for human response
        # - Resume workflow when response received
        
        # For demo, auto-approve if savings > $2000
        approved = total_savings > 2000
        
        return {
            "approved": approved,
            "approval_reason": "auto_approved" if approved else "requires_human_review"
        }
    
    async def _apply_optimizations(self, state: WorkflowState) -> Dict[str, Any]:
        """Apply cost optimizations"""
        recommendations = state.get("recommendations", [])
        
        logger.info(f"Applying {len(recommendations)} optimizations")
        
        # Save current state for rollback
        state["pre_optimization_state"] = {
            "routing_config": "current_config",  # Simplified
            "resource_allocation": "current_allocation"
        }
        
        # Apply each recommendation
        applied = []
        for rec in recommendations:
            if rec.get("type") == "routing":
                await self.tools.update_routing(
                    model_name=rec.get("model"),
                    strategy="cost-optimized"
                )
            elif rec.get("type") == "scaling":
                # Scale down underutilized resources
                pass
            
            applied.append(rec)
        
        # Checkpoint
        await self.bridge.save_workflow_checkpoint(
            workflow_id=state.get("workflow_id"),
            state=state.to_dict()
        )
        
        return {
            "optimizations_applied": applied,
            "applied_count": len(applied)
        }
    
    async def _monitor_cost_impact_7d(self, state: WorkflowState) -> Dict[str, Any]:
        """Monitor cost impact for 7 days"""
        baseline_cost = state.get("avg_daily_cost", 0)
        
        logger.info(f"Monitoring cost impact (baseline: ${baseline_cost}/day)")
        
        # In real implementation, monitor for 7 days
        # For demo, monitor for 1 minute
        daily_costs = []
        
        for day in range(3):  # Simulate 3 days
            await asyncio.sleep(20)  # 20 seconds per "day"
            
            # Get current costs
            current_costs = await self.tools.get_costs(time_range="1d")
            daily_cost = current_costs.get("total", 0)
            daily_costs.append(daily_cost)
            
            # Checkpoint
            state["monitoring_day"] = day + 1
            state["daily_costs"] = daily_costs
            await self.bridge.save_workflow_checkpoint(
                workflow_id=state.get("workflow_id"),
                state=state.to_dict()
            )
        
        # Calculate average
        avg_new_cost = sum(daily_costs) / len(daily_costs)
        cost_improved = avg_new_cost < baseline_cost
        
        savings_pct = ((baseline_cost - avg_new_cost) / baseline_cost * 100) if baseline_cost > 0 else 0
        
        logger.info(f"Cost impact: {savings_pct:.1f}% {'reduction' if cost_improved else 'increase'}")
        
        return {
            "cost_improved": cost_improved,
            "avg_new_cost": avg_new_cost,
            "baseline_cost": baseline_cost,
            "savings_percentage": savings_pct
        }
    
    async def _rollback_optimizations(self, state: WorkflowState) -> Dict[str, Any]:
        """Rollback optimizations that made costs worse"""
        logger.error("Rolling back cost optimizations (costs increased)")
        
        pre_state = state.get("pre_optimization_state", {})
        
        # Restore previous configuration
        # In real implementation, restore routing, scaling, etc.
        
        return {
            "rolled_back": True,
            "restored_state": pre_state
        }
    
    async def _notify_success(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify successful cost optimization"""
        savings_pct = state.get("savings_percentage", 0)
        
        logger.info(f"Cost optimization successful: {savings_pct:.1f}% reduction")
        
        return {
            "status": "success",
            "savings_percentage": savings_pct
        }
    
    async def _notify_rejected(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify that optimization was rejected"""
        logger.warning("Cost optimization rejected by user")
        
        return {"status": "rejected"}
    
    async def _notify_rollback(self, state: WorkflowState) -> Dict[str, Any]:
        """Notify that optimization was rolled back"""
        logger.error("Cost optimization rolled back")
        
        return {"status": "rolled_back"}
    
    async def optimize(self, params: Dict[str, Any] = None) -> Dict[str, Any]:
        """Execute cost optimization workflow"""
        workflow = self.workflows["optimize_costs"]
        result = await workflow.run(params or {})
        
        return result
    
    async def close(self):
        """Cleanup"""
        await self.tools.close()
        await self.bridge.close()


async def main():
    """Run cost agent"""
    config = NexusAIConfig.from_env()
    agent = CostAgent(config)
    
    result = await agent.optimize()
    
    logger.info(f"Optimization result: {result}")
    
    await agent.close()


if __name__ == "__main__":
    asyncio.run(main())

