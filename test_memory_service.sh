#!/bin/bash

set -euo pipefail

echo "🧠 TESTING: MEMORY SERVICES"
echo "============================"

echo ""
echo "1. WHAT: Check Memory Service deployment"
echo "   HOW: kubectl get deployment -n tensor-fusion-sys | grep memory"
kubectl get deployment -n tensor-fusion-sys | grep memory || echo "❌ Memory service not deployed"

echo ""
echo "2. WHAT: Test Memory Service health"
echo "   HOW: curl http://localhost:8090/health"
curl -s http://localhost:8090/health || echo "❌ Memory service health check failed"

echo ""
echo "3. WHAT: Store semantic memory"
echo "   HOW: curl -X POST http://localhost:8090/memory/semantic"
cat <<EOF | curl -X POST http://localhost:8090/memory/semantic \
  -H "Content-Type: application/json" \
  -d '{
    "key": "test_concept",
    "data": {
      "concept": "machine_learning",
      "description": "A method of data analysis that automates analytical model building",
      "related_concepts": ["artificial_intelligence", "statistics", "algorithms"]
    }
  }' || echo "❌ Semantic memory storage failed"
EOF

echo ""
echo "4. WHAT: Retrieve semantic memory"
echo "   HOW: curl http://localhost:8090/memory/semantic/test_concept"
curl -s http://localhost:8090/memory/semantic/test_concept || echo "❌ Semantic memory retrieval failed"

echo ""
echo "5. WHAT: Store episodic memory"
echo "   HOW: curl -X POST http://localhost:8090/memory/episodic"
cat <<EOF | curl -X POST http://localhost:8090/memory/episodic \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "training_session_001",
    "timestamp": "$(date -Iseconds)",
    "event_type": "model_training",
    "data": {
      "model": "llm-base",
      "dataset": "wikipedia",
      "accuracy": 0.85,
      "duration_minutes": 120
    }
  }' || echo "❌ Episodic memory storage failed"
EOF

echo ""
echo "6. WHAT: Search episodic memories"
echo "   HOW: curl http://localhost:8090/memory/episodic/search?event_type=model_training"
curl -s "http://localhost:8090/memory/episodic/search?event_type=model_training" || echo "❌ Episodic memory search failed"

echo ""
echo "7. WHAT: Test vector search (Qdrant)"
echo "   HOW: curl http://localhost:6333/health"
curl -s http://localhost:6333/health || echo "❌ Qdrant vector DB not accessible"

echo ""
echo "8. WHAT: Store memory in vector DB"
echo "   HOW: curl -X PUT http://localhost:6333/collections/test_memory/points"
cat <<EOF | curl -X PUT http://localhost:6333/collections/test_memory/points \
  -H "Content-Type: application/json" \
  -d '{
    "points": [
      {
        "id": 1,
        "vector": [0.1, 0.2, 0.3, 0.4, 0.5],
        "payload": {
          "text": "This is a test memory entry",
          "type": "semantic"
        }
      }
    ]
  }' || echo "❌ Vector memory storage failed"
EOF

echo ""
echo "9. WHAT: Search vector memory"
echo "   HOW: curl -X POST http://localhost:6333/collections/test_memory/points/search"
cat <<EOF | curl -X POST http://localhost:6333/collections/test_memory/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3, 0.4, 0.5],
    "limit": 5
  }' || echo "❌ Vector memory search failed"
EOF

echo ""
echo "10. WHAT: Test long-term memory persistence"
echo "    HOW: Check if memories persist after pod restart"
kubectl get pods -n tensor-fusion-sys | grep memory-service
echo "    Note: Would need to restart pod and re-check data persistence"

echo ""
echo "🎯 EXPECTED RESULTS:"
echo "• Memory service health check returns 200 OK"
echo "• Semantic and episodic memories stored successfully"
echo "• Memory retrieval works correctly"
echo "• Qdrant vector database accessible"
echo "• Vector search returns relevant results"
echo "• Memory persistence across pod restarts"

echo ""
echo "🧹 CLEANUP:"
echo "# Memory data persists - no cleanup needed for testing"

echo ""
echo "✅ MEMORY SERVICES TEST COMPLETE"
