#!/bin/bash
# Script to start BlackHole Blockchain services in sequence

echo "🚀 Starting BlackHole Blockchain Services..."

# Pre-flight Check: Verify Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker daemon is not running. Please start Docker and try again."
    exit 1
fi

# Pre-flight Check: Verify no port conflicts exist on the host
ports=(8080 8545 9000 8084 9090)
conflicts=()
for port in "${ports[@]}"; do
    if (echo >/dev/null 2>&1 </dev/tcp/127.0.0.1/$port); then
        conflicts+=($port)
    fi
done
if [ ${#conflicts[@]} -ne 0 ]; then
    echo "❌ Port conflict detected! The following ports are already in use: ${conflicts[*]}"
    echo "Please stop the processes using these ports before running the script."
    exit 1
fi

# Pre-flight Check: Ensure the external network exists
if ! docker network ls --filter name=blackhole-network -q | grep -q .; then
    echo "🌐 Creating docker network 'blackhole-network'..."
    docker network create blackhole-network >/dev/null
fi

# Stop existing containers first
echo ""
echo "📋 Stopping existing containers..."
docker compose down 2>&1 >/dev/null
docker compose -f docker-compose.blockchain.yml down 2>&1 >/dev/null
docker compose -f docker-compose.wallet.yml down 2>&1 >/dev/null
docker compose -f docker-compose.bridge.yml down 2>&1 >/dev/null

# Step 1: Start Blockchain nodes
echo ""
echo "⛓️  Step 1: Starting Blockchain nodes..."
docker compose -f docker-compose.blockchain.yml up -d
if [ $? -ne 0 ]; then
    echo "❌ Failed to start blockchain nodes"
    exit 1
fi

echo "⏳ Waiting for blockchain node to become healthy..."
max_wait=60
waited=0
healthy=false

while [ $waited -lt $max_wait ]; do
    status=$(docker inspect blackhole-node-1 --format='{{.State.Health.Status}}' 2>/dev/null)
    if [ "$status" = "healthy" ]; then
        healthy=true
        break
    fi
    sleep 2
    waited=$((waited + 2))
    echo -n "."
done
echo ""

if [ "$healthy" != "true" ]; then
    echo "❌ Blockchain node failed to become healthy within $max_wait seconds"
    echo "Check logs with: docker compose -f docker-compose.blockchain.yml logs"
    exit 1
fi

echo "✅ Blockchain nodes are healthy"

# Step 2: Start Wallet
echo ""
echo "💰 Step 2: Starting Wallet service..."
docker compose -f docker-compose.wallet.yml up -d
if [ $? -ne 0 ]; then
    echo "❌ Failed to start wallet service"
    exit 1
fi

echo "⏳ Waiting for wallet to become healthy (compiling dependencies)..."
max_wait=180
waited=0
healthy=false

while [ $waited -lt $max_wait ]; do
    status=$(docker inspect blackhole-wallet --format='{{.State.Health.Status}}' 2>/dev/null)
    if [ "$status" = "healthy" ]; then
        healthy=true
        break
    fi
    sleep 2
    waited=$((waited + 2))
    echo -n "."
done
echo ""

if [ "$healthy" != "true" ]; then
    echo "❌ Wallet service failed to become healthy within $max_wait seconds"
    echo "Check logs with: docker compose -f docker-compose.wallet.yml logs"
    exit 1
fi

echo "✅ Wallet service is healthy"

# Step 3: Start Bridge
echo ""
echo "🌉 Step 3: Starting Bridge service..."
docker compose -f docker-compose.bridge.yml up -d
if [ $? -ne 0 ]; then
    echo "❌ Failed to start bridge service"
    exit 1
fi

echo "⏳ Waiting for bridge to initialize..."
sleep 5

# Check final status
echo ""
echo "📊 Service Status:"
docker ps --filter "name=blackhole" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "✨ All services started successfully!"
echo ""
echo "📍 Service URLs:"
echo "  • Blockchain Node 1: http://localhost:8080"
echo "  • Wallet Dashboard:  http://localhost:9000"
echo "  • Bridge Dashboard:  http://localhost:8084"

echo ""
echo "💡 Useful commands:"
echo "  • Check logs:   docker compose -f docker-compose.<service>.yml logs -f <service>"
echo "  • Stop all:     docker compose down && docker compose -f docker-compose.blockchain.yml down && docker compose -f docker-compose.wallet.yml down && docker compose -f docker-compose.bridge.yml down"
echo "  • Restart all:  ./start-services.sh"
