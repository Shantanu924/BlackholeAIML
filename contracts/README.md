# BlackHole (BHIV) Token Contracts

Smart contracts for BlackHole Token (BHIV) deployment on Ethereum, BSC, and other EVM-compatible chains for DEX integration.

## 🚀 Quick Start

### 1. Install Dependencies
```bash
cd contracts
npm install
```

### 2. Configure Environment
```bash
cp .env.example .env
# Edit .env with your private key and RPC URLs
```

### 3. Compile Contracts
```bash
npm run compile
```

### 4. Deploy to Testnet (Recommended First)
```bash
# Deploy to BSC Testnet (cheaper for testing)
npm run deploy:bsc-testnet

# Deploy to Ethereum Sepolia
npm run deploy:sepolia
```

### 5. Deploy to Mainnet
```bash
# Deploy to BSC Mainnet (cheaper fees)
npm run deploy:bsc

# Deploy to Ethereum Mainnet
npm run deploy:ethereum
```

## 📋 Contract Features

### ✅ Standard ERC-20 Functionality
- Transfer, approve, transferFrom
- Standard events and interfaces
- Compatible with all wallets and DEXs

### ✅ Bridge Integration
- `bridgeMint()` - Mint tokens from other chains
- `bridgeBurn()` - Burn tokens for cross-chain transfer
- Bridge operator management
- Cross-chain event logging

### ✅ Security Features
- Pausable transfers (emergency stop)
- Blacklist functionality
- Anti-whale protection (configurable limits)
- Reentrancy protection
- Owner controls

### ✅ DEX Ready
- Standard ERC-20 interface
- No transfer fees (configurable)
- Liquidity pool compatible
- Trading pair ready

## 🎯 DEX Deployment Steps

### Step 1: Deploy Token Contract
```bash
# Deploy to BSC for PancakeSwap
npm run deploy:bsc

# Deploy to Ethereum for Uniswap
npm run deploy:ethereum
```

### Step 2: Verify Contract
```bash
# Verify on BSCScan
npm run verify:bsc <CONTRACT_ADDRESS>

# Verify on Etherscan
npm run verify:ethereum <CONTRACT_ADDRESS>
```

### Step 3: Add to DEX
1. Go to PancakeSwap/Uniswap
2. Import token using contract address
3. Add liquidity (ETH/BNB + BHIV)
4. Enable trading

## 💰 Deployment Costs

### BSC (Recommended for initial deployment)
- **Deployment**: ~$5-10 USD
- **Verification**: Free
- **Liquidity**: ~$20-50 USD in BNB

### Ethereum
- **Deployment**: ~$50-200 USD (depending on gas)
- **Verification**: Free
- **Liquidity**: ~$100-500 USD in ETH

## 🔧 Configuration

### Token Parameters
- **Name**: BlackHole Token
- **Symbol**: BHIV
- **Decimals**: 18
- **Initial Supply**: 100,000,000 BHIV
- **Max Supply**: 1,000,000,000 BHIV

### Anti-Whale Limits (Configurable)
- **Max Transaction**: 1,000,000 BHIV
- **Max Wallet**: 10,000,000 BHIV
- **Limits Enabled**: true (can be disabled)

## 🛡️ Security

### Access Control
- **Owner**: Can pause, blacklist, update limits
- **Bridge Operators**: Can mint/burn for cross-chain
- **Users**: Standard transfer functionality

### Emergency Functions
- Pause all transfers
- Blacklist malicious addresses
- Emergency token withdrawal
- Update transaction limits

## 📊 After Deployment

### 1. Configure Bridge Integration
```javascript
// Add your bridge contract as operator
await bhivToken.addBridgeOperator(BRIDGE_CONTRACT_ADDRESS);
```

### 2. Add Initial Liquidity
```javascript
// Example: Add 100,000 BHIV + 1 ETH to Uniswap
// Use DEX interface or router contract
```

### 3. Test Trading
- Small test transactions
- Verify bridge functionality
- Check wallet integration

## 🎯 DEX Integration Checklist

### Before Listing
- [ ] Contract deployed and verified
- [ ] Initial liquidity provided
- [ ] Bridge operators configured
- [ ] Security limits set
- [ ] Emergency functions tested

### PancakeSwap (BSC)
- [ ] Deploy to BSC
- [ ] Add BNB/BHIV liquidity
- [ ] Submit to PancakeSwap token list
- [ ] Test trading functionality

### Uniswap (Ethereum)
- [ ] Deploy to Ethereum
- [ ] Add ETH/BHIV liquidity
- [ ] Submit to Uniswap token list
- [ ] Test trading functionality

## 🚨 Important Notes

### Security
- **Never commit private keys to git**
- **Test on testnet first**
- **Verify contracts on block explorers**
- **Start with small liquidity amounts**

### Gas Optimization
- Deploy during low gas periods
- Use BSC for cheaper deployment
- Batch multiple operations

### Bridge Integration
- Configure bridge operators after deployment
- Test cross-chain functionality
- Monitor bridge transactions

## 📞 Support

For deployment issues or questions:
1. Check deployment logs in `deployments/` folder
2. Verify contract on block explorer
3. Test with small amounts first
4. Contact team for bridge integration

## 🎉 Success Metrics

After successful deployment, you should have:
- ✅ Verified contract on block explorer
- ✅ Trading pair on DEX
- ✅ Initial liquidity provided
- ✅ Bridge integration working
- ✅ Real users trading BHIV tokens

**Your BlackHole token is now ready for the world! 🌍**
