 #!/usr/bin/env pwsh
# Script to start BlackHole Blockchain services in sequence

Write-Host "🚀 Starting BlackHole Blockchain Services..." -ForegroundColor Cyan

# Pre-flight Check: Verify Docker is running
docker info >$null 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker is not running. Please start Docker Desktop and try again." -ForegroundColor Red
    exit 1
}

# Pre-flight Check: Verify no port conflicts exist on the host
$portsToCheck = @(8080, 8545, 9000, 8084, 9090)
$conflicts = @()
foreach ($port in $portsToCheck) {
    $conn = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    if ($conn) {
        $conflicts += $port
    }
}
if ($conflicts.Count -gt 0) {
    Write-Host "❌ Port conflict detected! The following ports are already in use on the host: $($conflicts -join ', ')" -ForegroundColor Red
    Write-Host "Please stop the processes using these ports before running the script." -ForegroundColor Yellow
    exit 1
}

# Pre-flight Check: Ensure the external network exists
$networkCheck = docker network ls --filter name=blackhole-network -q
if (-not $networkCheck) {
    Write-Host "🌐 Creating docker network 'blackhole-network'..." -ForegroundColor Yellow
    docker network create blackhole-network | Out-Null
}

# Stop existing containers first
Write-Host "`n📋 Stopping existing containers..." -ForegroundColor Yellow
docker compose down 2>&1 | Out-Null
docker compose -f docker-compose.blockchain.yml down 2>&1 | Out-Null
docker compose -f docker-compose.wallet.yml down 2>&1 | Out-Null
docker compose -f docker-compose.bridge.yml down 2>&1 | Out-Null

# Step 1: Start Blockchain nodes
Write-Host "`n⛓️  Step 1: Starting Blockchain nodes..." -ForegroundColor Green
docker compose -f docker-compose.blockchain.yml up -d
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to start blockchain nodes" -ForegroundColor Red
    exit 1
}

Write-Host "⏳ Waiting for blockchain node to become healthy..." -ForegroundColor Yellow
$maxWait = 60
$waited = 0
$healthy = $false

while ($waited -lt $maxWait) {
    $status = docker inspect blackhole-node-1 --format='{{.State.Health.Status}}' 2>$null
    if ($status -eq "healthy") {
        $healthy = $true
        break
    }
    Start-Sleep -Seconds 2
    $waited += 2
    Write-Host "." -NoNewline
}
Write-Host ""

if (-not $healthy) {
    Write-Host "❌ Blockchain node failed to become healthy within $maxWait seconds" -ForegroundColor Red
    Write-Host "Check logs with: docker compose -f docker-compose.blockchain.yml logs" -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Blockchain nodes are healthy" -ForegroundColor Green

# Step 2: Start Wallet
Write-Host "`n💰 Step 2: Starting Wallet service..." -ForegroundColor Green
docker compose -f docker-compose.wallet.yml up -d
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to start wallet service" -ForegroundColor Red
    exit 1
}

Write-Host "⏳ Waiting for wallet to become healthy (compiling dependencies)..." -ForegroundColor Yellow
$maxWait = 180
$waited = 0
$healthy = $false

while ($waited -lt $maxWait) {
    $status = docker inspect blackhole-wallet --format='{{.State.Health.Status}}' 2>$null
    if ($status -eq "healthy") {
        $healthy = $true
        break
    }
    Start-Sleep -Seconds 2
    $waited += 2
    Write-Host "." -NoNewline
}
Write-Host ""

if (-not $healthy) {
    Write-Host "❌ Wallet service failed to become healthy within $maxWait seconds" -ForegroundColor Red
    Write-Host "Check logs with: docker compose -f docker-compose.wallet.yml logs" -ForegroundColor Yellow
    exit 1
}

Write-Host "✅ Wallet service is healthy" -ForegroundColor Green

# Step 3: Start Bridge
Write-Host "`n🌉 Step 3: Starting Bridge service..." -ForegroundColor Green
docker compose -f docker-compose.bridge.yml up -d
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to start bridge service" -ForegroundColor Red
    exit 1
}

Write-Host "⏳ Waiting for bridge to initialize..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# Check final status
Write-Host "`n📊 Service Status:" -ForegroundColor Cyan
docker ps --filter "name=blackhole" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

Write-Host "`n✨ All services started successfully!" -ForegroundColor Green
Write-Host "`n📍 Service URLs:" -ForegroundColor Cyan
Write-Host "  • Blockchain Node 1: http://localhost:8080" -ForegroundColor White
Write-Host "  • Wallet Dashboard:  http://localhost:9000" -ForegroundColor White
Write-Host "  • Bridge Dashboard:  http://localhost:8084" -ForegroundColor White

Write-Host "`n💡 Useful commands:" -ForegroundColor Cyan
Write-Host "  • Check logs:   docker compose -f docker-compose.<service>.yml logs -f <service>" -ForegroundColor Gray
Write-Host "  • Stop all:     docker compose down && docker compose -f docker-compose.blockchain.yml down && docker compose -f docker-compose.wallet.yml down && docker compose -f docker-compose.bridge.yml down" -ForegroundColor Gray
Write-Host "  • Restart all:  .\start-services.ps1" -ForegroundColor Gray
