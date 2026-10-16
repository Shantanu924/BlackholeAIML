const { ethers, network } = require("hardhat");
const fs = require("fs");
const path = require("path");

async function main() {
    console.log("🚀 Deploying BHIV Token to", network.name);
    
    // Get deployer account
    const [deployer] = await ethers.getSigners();
    console.log("📝 Deploying with account:", deployer.address);
    console.log("💰 Account balance:", ethers.utils.formatEther(await deployer.getBalance()), "ETH");
    
    // Deployment parameters
    const INITIAL_SUPPLY = ethers.utils.parseEther("10000000"); // 10M BHIV for initial liquidity
    const MAX_SUPPLY = ethers.utils.parseEther("1000000000");   // 1B BHIV max supply
    
    console.log("📊 Token Parameters:");
    console.log("   Initial Supply:", ethers.utils.formatEther(INITIAL_SUPPLY), "BHIV");
    console.log("   Maximum Supply:", ethers.utils.formatEther(MAX_SUPPLY), "BHIV");
    
    // Deploy BHIV Token
    console.log("\n⏳ Deploying BHIV Token...");
    const BHIVToken = await ethers.getContractFactory("BHIVToken");
    const bhivToken = await BHIVToken.deploy(INITIAL_SUPPLY);
    await bhivToken.deployed();
    
    console.log("✅ BHIV Token deployed to:", bhivToken.address);
    console.log("🔗 Transaction hash:", bhivToken.deployTransaction.hash);
    
    // Verify deployment
    const tokenInfo = await bhivToken.getTokenInfo();
    console.log("\n📋 Deployed Token Info:");
    console.log("   Name:", tokenInfo.name);
    console.log("   Symbol:", tokenInfo.symbol);
    console.log("   Decimals:", tokenInfo.decimals);
    console.log("   Total Supply:", ethers.utils.formatEther(tokenInfo.totalSupply), "BHIV");
    console.log("   Max Supply:", ethers.utils.formatEther(tokenInfo.maxSupply), "BHIV");
    console.log("   Owner:", await bhivToken.owner());
    
    // Save deployment info
    const deploymentInfo = {
        network: network.name,
        chainId: network.config.chainId,
        contractAddress: bhivToken.address,
        deployerAddress: deployer.address,
        transactionHash: bhivToken.deployTransaction.hash,
        blockNumber: bhivToken.deployTransaction.blockNumber,
        gasUsed: bhivToken.deployTransaction.gasLimit.toString(),
        timestamp: new Date().toISOString(),
        tokenInfo: {
            name: tokenInfo.name,
            symbol: tokenInfo.symbol,
            decimals: tokenInfo.decimals,
            initialSupply: ethers.utils.formatEther(tokenInfo.totalSupply),
            maxSupply: ethers.utils.formatEther(tokenInfo.maxSupply)
        }
    };
    
    // Create deployments directory if it doesn't exist
    const deploymentsDir = path.join(__dirname, "../deployments");
    if (!fs.existsSync(deploymentsDir)) {
        fs.mkdirSync(deploymentsDir, { recursive: true });
    }
    
    // Save deployment info to file
    const deploymentFile = path.join(deploymentsDir, `bhiv-${network.name}.json`);
    fs.writeFileSync(deploymentFile, JSON.stringify(deploymentInfo, null, 2));
    console.log("💾 Deployment info saved to:", deploymentFile);
    
    // Generate exchange submission data
    const exchangeData = {
        tokenName: "BlackHole",
        tokenSymbol: "BHIV",
        contractAddress: bhivToken.address,
        decimals: 18,
        totalSupply: "1000000000",
        website: "https://blackhole-blockchain.com",
        whitepaper: "https://blackhole-blockchain.com/whitepaper.pdf",
        socialLinks: {
            twitter: "https://twitter.com/BlackHoleChain",
            telegram: "https://t.me/BlackHoleChain",
            discord: "https://discord.gg/BlackHoleChain"
        },
        description: "BlackHole (BHIV) is the native token of BlackHole Blockchain, a high-performance blockchain with built-in cross-chain bridge, DEX, and DeFi features.",
        logoUrl: "https://blackhole-blockchain.com/logo.png",
        networkInfo: {
            blockchain: "Ethereum",
            network: network.name,
            chainId: network.config.chainId
        }
    };
    
    const exchangeFile = path.join(deploymentsDir, `bhiv-exchange-data.json`);
    fs.writeFileSync(exchangeFile, JSON.stringify(exchangeData, null, 2));
    console.log("📈 Exchange submission data saved to:", exchangeFile);
    
    // Verification instructions
    console.log("\n🔍 Contract Verification:");
    console.log("Run the following command to verify on Etherscan:");
    console.log(`npx hardhat verify --network ${network.name} ${bhivToken.address} "${INITIAL_SUPPLY}"`);
    
    // Next steps
    console.log("\n📋 Next Steps for Exchange Listing:");
    console.log("1. ✅ Contract deployed successfully");
    console.log("2. 🔍 Verify contract on Etherscan");
    console.log("3. 💧 Add liquidity to Uniswap V3");
    console.log("4. 📊 Submit to CoinGecko & CoinMarketCap");
    console.log("5. 📧 Apply to centralized exchanges");
    console.log("6. 🌉 Deploy bridge contracts");
    
    return {
        bhivToken: bhivToken.address,
        deployer: deployer.address,
        network: network.name
    };
}

// We recommend this pattern to be able to use async/await everywhere
// and properly handle errors.
main()
    .then((result) => {
        console.log("\n🎉 Deployment completed successfully!");
        console.log("Contract Address:", result.bhivToken);
        process.exit(0);
    })
    .catch((error) => {
        console.error("❌ Deployment failed:", error);
        process.exit(1);
    });