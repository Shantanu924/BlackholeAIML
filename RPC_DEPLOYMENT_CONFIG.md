# VAJRA RPC Node: Production Deployment Parameters

To support `rpc.vajra.blackholeinfiverse.com`, the following deployment topology and security parameters must be applied to the public-facing Relay Node.

## 1. Network & Port Configuration

The RPC Node must securely expose the following ports:
- **HTTP JSON-RPC API (Port 8080):** Externally accessible via Nginx reverse proxy. Used by Wallets, Exchanges, and web3 clients.
- **gRPC API (Port 50051):** Externally accessible. Used for high-throughput institutional trading or internal microservices.
- **P2P Libp2p (Port 3000):** Externally accessible. Required for blockchain peer discovery and block gossiping.

## 2. Nginx Reverse Proxy (rpc.vajra.blackholeinfiverse.com)
The node should NEVER be directly exposed to the internet. Traffic must route through an Nginx proxy with the following configuration:

```nginx
server {
    listen 443 ssl http2;
    server_name rpc.vajra.blackholeinfiverse.com;

    # SSL Certificates (Let's Encrypt / AWS ACM)
    ssl_certificate /etc/letsencrypt/live/rpc.vajra.blackholeinfiverse.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/rpc.vajra.blackholeinfiverse.com/privkey.pem;

    # Rate Limiting (Prevent DDoS)
    limit_req zone=rpc_limit burst=50 nodelay;

    location / {
        proxy_pass http://localhost:8080; # Internal Relay API
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        
        # CORS Headers (Crucial for Wallet UI to connect)
        add_header 'Access-Control-Allow-Origin' '*';
        add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
        add_header 'Access-Control-Allow-Headers' 'DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization';
    }
}
```

## 3. Hardware Requirements (AWS EC2 / DigitalOcean)
- **Instance Type:** Compute Optimized (e.g., AWS c6g.xlarge / DO Premium Intel 4 vCPUs)
- **RAM:** 8 GB minimum (16 GB recommended for high-volume RPC)
- **Storage:** 500 GB NVMe SSD (Blockchain state grows rapidly; IOPS are critical for fast queries).

## 4. Environment Configuration (`.env`)
The RPC Node container requires the following environment variables:
```env
# Node Identity
NODE_ROLE=RPC_GATEWAY
NETWORK_ID=vajra-mainnet

# Security
ENABLE_CORS=true
ALLOWED_ORIGINS=*
RATE_LIMIT_RPS=100

# Blockchain
P2P_PORT=3000
API_PORT=8080
GRPC_PORT=50051
```

## 5. DDoS & Firewall Protection
- **Cloudflare / AWS Shield:** Route `rpc.vajra.blackholeinfiverse.com` through Cloudflare with "Under Attack" mode ready.
- **Firewall:** Drop all incoming connections except 443 (HTTPS), 3000 (TCP/UDP P2P). Port 8080 should ONLY be accessible internally by Nginx.
