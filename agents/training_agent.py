"""
NexusAI Training Agent - Microsoft Agent Framework Implementation

Specialized agent for model training with:
- Automatic checkpointing every N steps
- Resume from failure
- Hyperparameter auto-tuning
- Parallel training trials
"""

import asyncio
from typing import Dict, Any, List, Optional
from agent_framework.workflows import GraphWorkflow
from agent_framework.workflows.state import WorkflowState
from agent_framework.logging import get_logger

from nexusai import NexusAIMCPTools, NexusAIConfig, WorkflowBridge

logger = get_logger(__name__)


class TrainingAgent:
    """
    Specialized Training Agent using Microsoft Agent Framework.
    
    Key features:
    - Automatic checkpointing every 10% of training
    - Resume training from any checkpoint
    - Parallel hyperparameter tuning
    - Auto-retry on failure
    - Quality gates (accuracy thresholds)
    """
    
    def __init__(self, config: NexusAIConfig):
        self.config = config
        self.tools = NexusAIMCPTools(config)
        self.bridge = WorkflowBridge(config)
        self.workflows = {}
        
        self._register_workflows()
    
    def _register_workflows(self):
        """Register training workflows"""
        self.workflows["train_lora"] = self.create_lora_training_workflow()
        self.workflows["train_with_hpo"] = self.create_hyperparameter_optimization_workflow()
    
    # ========== WORKFLOW 1: LoRA Training with Checkpointing ==========
    
    def create_lora_training_workflow(self) -> GraphWorkflow:
        """
        LoRA Training Workflow with automatic checkpointing.
        
        Graph:
        validate_input → prepare_dataset → allocate_resources → start_training → monitor_checkpoints
                                                                                → [complete] → validate_model → [quality OK?]
                                                                                                              → [YES] → save_model
                                                                                                              → [NO] → retry_with_tuning
                                                                                → [failed] → resume_from_checkpoint
        """
        workflow = GraphWorkflow(name="train_lora")
        
        # Add nodes
        workflow.add_node("validate_input", self._validate_training_input)
        workflow.add_node("prepare_dataset", self._prepare_dataset)
        workflow.add_node("allocate_resources", self._allocate_training_resources)
        workflow.add_node("start_training", self._start_training_job)
        workflow.add_node("monitor_checkpoints", self._monitor_with_checkpoints)
        workflow.add_node("validate_model", self._validate_trained_model)
        workflow.add_node("save_model", self._save_final_model)
        workflow.add_node("retry_with_tuning", self._retry_with_adjusted_hyperparameters)
        workflow.add_node("resume_from_checkpoint", self._resume_training)
        
        # Add edges
        workflow.add_edge("validate_input", "prepare_dataset")
        workflow.add_edge("prepare_dataset", "allocate_resources")
        workflow.add_edge("allocate_resources", "start_training")
        workflow.add_edge("start_training", "monitor_checkpoints")
        
        # Conditional: training complete or failed?
        workflow.add_conditional_edge(
            "monitor_checkpoints",
            self._check_training_status,
            {
                "complete": "validate_model",
                "failed": "resume_from_checkpoint"
            }
        )
        
        workflow.add_edge("resume_from_checkpoint", "monitor_checkpoints")
        
        # Conditional: model quality sufficient?
        workflow.add_conditional_edge(
            "validate_model",
            self._check_model_quality,
            {
                "pass": "save_model",
                "fail": "retry_with_tuning"
            }
        )
        
        workflow.add_edge("retry_with_tuning", "start_training")
        
        workflow.set_entry_point("validate_input")
        workflow.set_exit_point("save_model")
        
        # Enable checkpointing (critical for training!)
        workflow.enable_checkpointing()
        
        return workflow
    
    # ========== Node Implementations ==========
    
    async def _validate_training_input(self, state: WorkflowState) -> Dict[str, Any]:
        """Validate training parameters"""
        base_model = state.get("base_model")
        dataset_path = state.get("dataset_path")
        
        if not base_model or not dataset_path:
            raise ValueError("base_model and dataset_path are required")
        
        logger.info(f"Validating training: {base_model} with {dataset_path}")
        
        # Check dataset exists
        # In real implementation, verify file accessibility
        
        return {
            "valid": True,
            "base_model": base_model,
            "dataset_path": dataset_path,
            "training_type": state.get("training_type", "lora")
        }
    
    async def _prepare_dataset(self, state: WorkflowState) -> Dict[str, Any]:
        """Prepare dataset for training"""
        dataset_path = state.get("dataset_path")
        
        logger.info(f"Preparing dataset: {dataset_path}")
        
        # In real implementation:
        # - Validate format (JSONL, Parquet, etc.)
        # - Split train/val/test
        # - Tokenize
        # - Upload to shared storage
        
        return {
            "prepared": True,
            "train_samples": 4000,
            "val_samples": 500,
            "test_samples": 500,
            "prepared_path": f"/data/prepared/{dataset_path.split('/')[-1]}"
        }
    
    async def _allocate_training_resources(self, state: WorkflowState) -> Dict[str, Any]:
        """Allocate GPU resources for training"""
        # Calculate required GPU based on model size
        base_model = state.get("base_model")
        
        # Model size estimation (simplified)
        vgpu_required = 2.0  # For LoRA training
        if "70b" in base_model.lower():
            vgpu_required = 4.0
        elif "13b" in base_model.lower():
            vgpu_required = 2.0
        else:
            vgpu_required = 1.0
        
        logger.info(f"Allocating {vgpu_required} vGPU for training")
        
        result = await self.tools.allocate_gpu(
            vgpu_size=vgpu_required,
            duration="48h",  # Longer for training
            pool_name="training-pool"
        )
        
        return {
            "allocated": True,
            "vgpu_size": vgpu_required,
            "allocation_id": result.get("allocation_id")
        }
    
    async def _start_training_job(self, state: WorkflowState) -> Dict[str, Any]:
        """Start the training job"""
        base_model = state.get("base_model")
        dataset_path = state.get("prepared_path")
        training_type = state.get("training_type", "lora")
        
        # Get hyperparameters (use defaults or provided)
        hyperparameters = state.get("hyperparameters", {
            "learning_rate": 2e-4,
            "num_epochs": 3,
            "batch_size": 4,
            "lora_r": 8,
            "lora_alpha": 16,
            "lora_dropout": 0.05
        })
        
        logger.info(f"Starting training: {base_model}")
        
        result = await self.tools.start_training(
            base_model=base_model,
            dataset_path=dataset_path,
            training_type=training_type,
            **hyperparameters
        )
        
        job_id = result.get("job_id")
        
        # Save checkpoint immediately after starting
        state["job_id"] = job_id
        state["training_started_at"] = result.get("started_at")
        await self.bridge.save_workflow_checkpoint(
            workflow_id=state.get("workflow_id"),
            state=state.to_dict()
        )
        
        return {
            "job_id": job_id,
            "status": "running",
            "started_at": result.get("started_at")
        }
    
    async def _monitor_with_checkpoints(self, state: WorkflowState) -> Dict[str, Any]:
        """
        Monitor training progress with automatic checkpointing.
        
        Saves checkpoint every 10% of progress.
        """
        job_id = state.get("job_id")
        last_checkpoint = state.get("last_progress", 0)
        
        logger.info(f"Monitoring training job: {job_id}")
        
        while True:
            # Check status
            status = await self.tools.get_training_status(job_id)
            
            current_status = status.get("status")
            progress = status.get("progress", 0)  # 0-100
            current_loss = status.get("loss", 0)
            current_accuracy = status.get("accuracy", 0)
            
            # Update state
            state["status"] = current_status
            state["progress"] = progress
            state["loss"] = current_loss
            state["accuracy"] = current_accuracy
            
            # Save checkpoint every 10%
            if progress - last_checkpoint >= 10:
                logger.info(f"Saving checkpoint at {progress}% progress")
                await self.bridge.save_workflow_checkpoint(
                    workflow_id=state.get("workflow_id"),
                    state=state.to_dict()
                )
                state["last_progress"] = progress
                last_checkpoint = progress
            
            # Check if complete or failed
            if current_status in ["completed", "failed", "error"]:
                # Final checkpoint
                await self.bridge.save_workflow_checkpoint(
                    workflow_id=state.get("workflow_id"),
                    state=state.to_dict()
                )
                break
            
            # Wait before next check
            await asyncio.sleep(30)
        
        return {
            "status": current_status,
            "final_progress": progress,
            "final_loss": current_loss,
            "final_accuracy": current_accuracy
        }
    
    def _check_training_status(self, state: WorkflowState) -> str:
        """Check if training completed or failed"""
        status = state.get("status")
        return "complete" if status == "completed" else "failed"
    
    async def _resume_training(self, state: WorkflowState) -> Dict[str, Any]:
        """Resume training from last checkpoint"""
        job_id = state.get("job_id")
        last_progress = state.get("last_progress", 0)
        
        logger.warning(f"Training failed at {last_progress}%. Resuming from checkpoint...")
        
        # In real implementation:
        # - Load checkpoint weights
        # - Resume training from that step
        # - Use same job_id to continue
        
        # For now, restart the monitoring
        state["status"] = "resuming"
        
        return {
            "resumed": True,
            "resume_from": last_progress
        }
    
    async def _validate_trained_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Validate the trained model quality"""
        job_id = state.get("job_id")
        final_accuracy = state.get("final_accuracy", 0)
        final_loss = state.get("final_loss", float('inf'))
        
        logger.info(f"Validating model: accuracy={final_accuracy}, loss={final_loss}")
        
        # Quality gates
        accuracy_threshold = state.get("accuracy_threshold", 0.85)
        quality_pass = final_accuracy >= accuracy_threshold
        
        return {
            "quality_pass": quality_pass,
            "accuracy": final_accuracy,
            "loss": final_loss,
            "threshold": accuracy_threshold
        }
    
    def _check_model_quality(self, state: WorkflowState) -> str:
        """Check if model meets quality threshold"""
        return "pass" if state.get("quality_pass") else "fail"
    
    async def _retry_with_adjusted_hyperparameters(self, state: WorkflowState) -> Dict[str, Any]:
        """Retry training with adjusted hyperparameters"""
        retry_count = state.get("retry_count", 0)
        
        if retry_count >= 3:
            logger.error("Max retries reached. Training failed.")
            raise Exception("Training failed after 3 attempts")
        
        logger.warning(f"Retrying training (attempt {retry_count + 1}/3)")
        
        # Adjust hyperparameters
        current_hp = state.get("hyperparameters", {})
        
        # Strategy: reduce learning rate, increase epochs
        current_hp["learning_rate"] = current_hp.get("learning_rate", 2e-4) * 0.5
        current_hp["num_epochs"] = current_hp.get("num_epochs", 3) + 1
        
        state["hyperparameters"] = current_hp
        state["retry_count"] = retry_count + 1
        
        return {
            "retrying": True,
            "attempt": retry_count + 1,
            "adjusted_hyperparameters": current_hp
        }
    
    async def _save_final_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Save the trained model"""
        job_id = state.get("job_id")
        base_model = state.get("base_model")
        accuracy = state.get("accuracy")
        
        logger.info(f"Saving trained model: {base_model} (accuracy: {accuracy})")
        
        # In real implementation:
        # - Save model weights to MinIO/Blob Storage
        # - Register in Model Catalog
        # - Create TrainedModel CR
        
        model_path = f"/models/trained/{base_model}-{job_id}"
        
        return {
            "status": "saved",
            "model_path": model_path,
            "job_id": job_id,
            "accuracy": accuracy
        }
    
    # ========== WORKFLOW 2: Hyperparameter Optimization ==========
    
    def create_hyperparameter_optimization_workflow(self) -> GraphWorkflow:
        """
        Hyperparameter Optimization Workflow.
        
        Tries multiple hyperparameter combinations in parallel,
        picks the best one.
        
        Graph:
        generate_hp_configs → train_parallel → compare_results → save_best_model
        """
        workflow = GraphWorkflow(name="train_with_hpo")
        
        workflow.add_node("generate_hp_configs", self._generate_hyperparameter_configs)
        workflow.add_node("train_parallel", self._train_parallel_trials)
        workflow.add_node("compare_results", self._compare_hp_results)
        workflow.add_node("save_best", self._save_best_hpo_model)
        
        workflow.add_edge("generate_hp_configs", "train_parallel")
        workflow.add_edge("train_parallel", "compare_results")
        workflow.add_edge("compare_results", "save_best")
        
        workflow.set_entry_point("generate_hp_configs")
        workflow.set_exit_point("save_best")
        
        workflow.enable_checkpointing()
        
        return workflow
    
    async def _generate_hyperparameter_configs(self, state: WorkflowState) -> Dict[str, Any]:
        """Generate hyperparameter configurations to try"""
        num_trials = state.get("num_trials", 3)
        
        logger.info(f"Generating {num_trials} HP configs")
        
        # Generate configs (simplified)
        configs = [
            {"learning_rate": 1e-4, "batch_size": 4, "lora_r": 8},
            {"learning_rate": 2e-4, "batch_size": 4, "lora_r": 16},
            {"learning_rate": 5e-4, "batch_size": 8, "lora_r": 8},
        ][:num_trials]
        
        return {
            "hp_configs": configs
        }
    
    async def _train_parallel_trials(self, state: WorkflowState) -> Dict[str, Any]:
        """Train multiple trials in parallel"""
        configs = state.get("hp_configs", [])
        base_model = state.get("base_model")
        dataset_path = state.get("dataset_path")
        
        logger.info(f"Starting {len(configs)} parallel training trials")
        
        # Start all trials
        trials = []
        for i, config in enumerate(configs):
            result = await self.tools.start_training(
                base_model=base_model,
                dataset_path=dataset_path,
                training_type="lora",
                **config
            )
            
            trials.append({
                "trial_id": i,
                "job_id": result.get("job_id"),
                "config": config,
                "status": "running"
            })
        
        # Monitor all trials
        completed_trials = await self._monitor_parallel_trials(trials, state)
        
        return {
            "trials": completed_trials
        }
    
    async def _monitor_parallel_trials(
        self,
        trials: List[Dict[str, Any]],
        state: WorkflowState
    ) -> List[Dict[str, Any]]:
        """Monitor multiple training trials"""
        while True:
            all_complete = True
            
            for trial in trials:
                if trial["status"] in ["completed", "failed"]:
                    continue
                
                status = await self.tools.get_training_status(trial["job_id"])
                trial["status"] = status.get("status")
                trial["accuracy"] = status.get("accuracy", 0)
                trial["loss"] = status.get("loss", 0)
                
                if trial["status"] not in ["completed", "failed"]:
                    all_complete = False
            
            # Save checkpoint
            state["trials"] = trials
            await self.bridge.save_workflow_checkpoint(
                workflow_id=state.get("workflow_id"),
                state=state.to_dict()
            )
            
            if all_complete:
                break
            
            await asyncio.sleep(60)  # Check every minute
        
        return trials
    
    async def _compare_hp_results(self, state: WorkflowState) -> Dict[str, Any]:
        """Compare HPO results"""
        trials = state.get("trials", [])
        
        # Find best by accuracy
        best = max(trials, key=lambda x: x.get("accuracy", 0))
        
        logger.info(f"Best trial: {best['trial_id']} with accuracy {best['accuracy']}")
        
        return {
            "best_trial": best,
            "best_config": best["config"],
            "best_accuracy": best["accuracy"]
        }
    
    async def _save_best_hpo_model(self, state: WorkflowState) -> Dict[str, Any]:
        """Save the best model from HPO"""
        best_trial = state.get("best_trial")
        
        logger.info(f"Saving best HPO model: {best_trial['job_id']}")
        
        return {
            "status": "saved",
            "model_path": f"/models/hpo/{best_trial['job_id']}",
            "config": best_trial["config"],
            "accuracy": best_trial["accuracy"]
        }
    
    # ========== Public API ==========
    
    async def train(
        self,
        workflow_name: str,
        params: Dict[str, Any],
        workflow_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Execute a training workflow.
        
        Supports:
        - Resume from checkpoint
        - Auto-retry on failure
        - Parallel trials (for HPO)
        """
        if workflow_name not in self.workflows:
            raise ValueError(f"Unknown workflow: {workflow_name}")
        
        workflow = self.workflows[workflow_name]
        
        # Check for checkpoint
        if workflow_id:
            checkpoint = await self.bridge.load_workflow_checkpoint(workflow_id)
            if checkpoint:
                logger.info(f"Resuming training from checkpoint")
                params.update(checkpoint)
        
        # Execute
        result = await workflow.run(params)
        
        return result
    
    async def close(self):
        """Cleanup"""
        await self.tools.close()
        await self.bridge.close()


# ========== Main ==========

async def main():
    """Run training agent"""
    config = NexusAIConfig.from_env()
    agent = TrainingAgent(config)
    
    # Example: Train LoRA
    result = await agent.train(
        workflow_name="train_lora",
        params={
            "base_model": "llama-3.1-8b",
            "dataset_path": "/data/my-dataset.jsonl",
            "accuracy_threshold": 0.90
        }
    )
    
    logger.info(f"Training result: {result}")
    
    await agent.close()


if __name__ == "__main__":
    asyncio.run(main())

