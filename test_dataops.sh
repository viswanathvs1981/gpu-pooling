#!/bin/bash

set -euo pipefail

echo "📊 TESTING: DATAOPS PIPELINE"
echo "============================"

echo ""
echo "1. WHAT: Check DataOps agents deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep dataops"
kubectl get deployment -n tensor-fusion-sys | grep dataops || echo "❌ DataOps agents not deployed"

echo ""
echo "2. WHAT: Test DataOps agents health"
echo "   HOW: curl http://localhost:8083/health"
curl -s http://localhost:8083/health || echo "❌ DataOps health check failed"

echo ""
echo "3. WHAT: Test data pipeline creation"
echo "   HOW: curl -X POST http://localhost:8083/pipelines"
cat <<EOF | curl -X POST http://localhost:8083/pipelines \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-etl-pipeline",
    "type": "etl",
    "source": "postgresql://test-db",
    "destination": "minio://test-bucket",
    "transformations": ["validate", "clean", "aggregate"]
  }' || echo "❌ Pipeline creation failed"
EOF

echo ""
echo "4. WHAT: Test feature engineering"
echo "   HOW: curl -X POST http://localhost:8083/features/generate"
cat <<EOF | curl -X POST http://localhost:8083/features/generate \
  -H "Content-Type: application/json" \
  -d '{
    "dataset": "test_dataset",
    "target_column": "price",
    "feature_types": ["numerical", "categorical"],
    "max_features": 10
  }' || echo "❌ Feature engineering failed"
EOF

echo ""
echo "5. WHAT: Test drift detection"
echo "   HOW: curl -X POST http://localhost:8083/drift/detect"
cat <<EOF | curl -X POST http://localhost:8083/drift/detect \
  -H "Content-Type: application/json" \
  -d '{
    "model_id": "test-model",
    "current_data": [[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]],
    "reference_data": [[1.1, 2.1, 3.1], [4.1, 5.1, 6.1]]
  }' || echo "❌ Drift detection failed"
EOF

echo ""
echo "6. WHAT: Test data lineage tracking"
echo "   HOW: curl -X POST http://localhost:8083/lineage/track"
cat <<EOF | curl -X POST http://localhost:8083/lineage/track \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "data_transformation",
    "input_datasets": ["raw_sales.csv"],
    "output_datasets": ["cleaned_sales.parquet"],
    "transformations": ["remove_nulls", "normalize_prices"]
  }' || echo "❌ Lineage tracking failed"
EOF

echo ""
echo "7. WHAT: Test experiment tracking"
echo "   HOW: curl -X POST http://localhost:8083/experiments"
cat <<EOF | curl -X POST http://localhost:8083/experiments \
  -H "Content-Type: application/json" \
  -d '{
    "experiment_name": "model_tuning_test",
    "model_type": "xgboost",
    "parameters": {
      "learning_rate": 0.01,
      "max_depth": 6,
      "n_estimators": 100
    },
    "metrics": {
      "accuracy": 0.85,
      "precision": 0.82,
      "recall": 0.88
    }
  }' || echo "❌ Experiment tracking failed"
EOF

echo ""
echo "8. WHAT: Check experiment results"
echo "   HOW: curl http://localhost:8083/experiments"
curl -s http://localhost:8083/experiments | head -5 || echo "❌ Cannot retrieve experiments"

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• DataOps agents deployment running"
echo "• Health checks return 200 OK"
echo "• ETL pipelines created successfully"
echo "• Feature engineering generates features"
echo "• Drift detection identifies changes"
echo "• Data lineage tracked correctly"
echo "• Experiments logged with metrics"

echo ""
echo "🧹 CLEANUP:"
echo "# Test data persists - cleanup if needed"

echo ""
echo "✅ DATAOPS PIPELINE TEST COMPLETE"
