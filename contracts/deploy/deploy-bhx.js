const { ethers } = require("hardhat");

async function main() {
    console.log("🚀 Deploying BHIV Token Contract...");
    
    // Get deployer account
    const [deployer] = await ethers.getSigners();
    console.log("📝 Deploying with account:", deployer.address);
    
    // Check balance
    const balance = await deployer.getBalance();
    console.log("💰 Account balance:", ethers.utils.formatEther(balance), "ETH");
    
    // Deploy BHIV Token
    console.log("\n🔨 Deploying BHIVToken...");
    const BHIVToken = await ethers.getContractFactory("BHIVToken");
    const bhivToken = await BHIVToken.deploy();
    
    await bhivToken.deployed();
    
    console.log("✅ BHIVToken deployed to:", bhivToken.address);
    console.log("🔗 Transaction hash:", bhivToken.deployTransaction.hash);
    
    // Wait for confirmations
    console.log("⏳ Waiting for confirmations...");
    await bhivToken.deployTransaction.wait(5);
    
    // Get token info
    const tokenInfo = await bhivToken.getTokenInfo();
    console.log("\n📊 Token Information:");
    console.log("   Name:", tokenInfo.name);
    console.log("   Symbol:", tokenInfo.symbol);
    console.log("   Decimals:", tokenInfo.decimals);
    console.log("   Total Supply:", ethers.utils.formatEther(tokenInfo.totalSupply), "BHIV");
    console.log("   Max Supply:", ethers.utils.formatEther(tokenInfo.maxSupply), "BHIV");
    console.log("   Paused:", tokenInfo.paused);
    console.log("   Limits Enabled:", tokenInfo.limitsEnabled);
    
    // Verify deployer is bridge operator
    const isBridgeOperator = await bhivToken.isBridgeOperator(deployer.address);
    console.log("   Deployer is Bridge Operator:", isBridgeOperator);
    
    console.log("\n🎉 Deployment completed successfully!");
    console.log("\n📋 Next Steps:");
    console.log("1. Verify contract on block explorer");
    console.log("2. Add to DEX (PancakeSwap/Uniswap)");
    console.log("3. Provide initial liquidity");
    console.log("4. Configure bridge operators");
    
    // Save deployment info
    const deploymentInfo = {
        network: network.name,
        contractAddress: bhivToken.address,
        deployerAddress: deployer.address,
        transactionHash: bhivToken.deployTransaction.hash,
        blockNumber: bhivToken.deployTransaction.blockNumber,
        timestamp: new Date().toISOString(),
        tokenInfo: {
            name: tokenInfo.name,
            symbol: tokenInfo.symbol,
            decimals: tokenInfo.decimals.toString(),
            totalSupply: tokenInfo.totalSupply.toString(),
            maxSupply: tokenInfo.maxSupply.toString()
        }
    };
    
    const fs = require('fs');
    const deploymentPath = `deployments/${network.name}-bhiv-deployment.json`;
    
    // Create deployments directory if it doesn't exist
    if (!fs.existsSync('deployments')) {
        fs.mkdirSync('deployments');
    }
    
    fs.writeFileSync(deploymentPath, JSON.stringify(deploymentInfo, null, 2));
    console.log(`💾 Deployment info saved to: ${deploymentPath}`);
    
    return {
        contract: bhivToken,
        address: bhivToken.address,
        deploymentInfo
    };
}

// Handle deployment
if (require.main === module) {
    main()
        .then(() => process.exit(0))
        .catch((error) => {
            console.error("❌ Deployment failed:", error);
            process.exit(1);
        });
}

module.exports = main;
