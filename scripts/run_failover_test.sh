#!/bin/bash

# run_failover_test.sh
# Tests the resilience of the BlackHole validator network during a node failure.

echo "=========================================="
echo "    BlackHole Validator Failover Test     "
echo "=========================================="

echo "[1/4] Building Relay Chain binary..."
cd ../core/relay-chain/cmd
go build -o relay_node .
cd ../../../scripts

echo "[2/4] Starting validator nodes..."
# Start Node 1 (Genesis / Main)
../core/relay-chain/cmd/relay_node -p2p 3030 -api 8080 -validator "genesis-validator-1" > node1.log 2>&1 &
NODE1_PID=$!
echo "  -> Node 1 started (PID: $NODE1_PID, P2P: 3030)"
sleep 3

# Start Node 2
../core/relay-chain/cmd/relay_node -p2p 3031 -api 8081 -peer "/ip4/127.0.0.1/tcp/3030/p2p/node1_id_placeholder" -validator "genesis-validator-2" > node2.log 2>&1 &
NODE2_PID=$!
echo "  -> Node 2 started (PID: $NODE2_PID, P2P: 3031)"
sleep 3

# Start Node 3
../core/relay-chain/cmd/relay_node -p2p 3032 -api 8082 -peer "/ip4/127.0.0.1/tcp/3030/p2p/node1_id_placeholder" -validator "genesis-validator-3" > node3.log 2>&1 &
NODE3_PID=$!
echo "  -> Node 3 started (PID: $NODE3_PID, P2P: 3032)"
sleep 5

echo "[3/4] Simulating Node 1 failure (Hard kill)..."
kill -9 $NODE1_PID
echo "  -> Node 1 killed."

echo "[4/4] Monitoring network recovery..."
sleep 10
# Check if Node 2 and Node 3 are still alive
if ps -p $NODE2_PID > /dev/null; then
   echo "✅ Node 2 survived the failover."
else
   echo "❌ Node 2 crashed!"
fi

if ps -p $NODE3_PID > /dev/null; then
   echo "✅ Node 3 survived the failover."
else
   echo "❌ Node 3 crashed!"
fi

echo "Cleaning up..."
kill -9 $NODE2_PID 2>/dev/null
kill -9 $NODE3_PID 2>/dev/null

echo "=========================================="
echo "          Failover Test Complete          "
echo "=========================================="
