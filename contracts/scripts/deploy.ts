import { ethers } from "hardhat";
import { CredToken, Settlement } from "../typechain-types";

async function main() {
  const [deployer] = await ethers.getSigners();
  console.log("Deploying contracts with account:", deployer.address);
  console.log("Account balance:", (await deployer.provider.getBalance(deployer.address)).toString());

  // Get network
  const network = await ethers.provider.getNetwork();
  console.log("Deploying to network:", network.name, "(chainId:", network.chainId, ")");

  // Deploy CredToken
  console.log("\nDeploying CredToken...");
  const CredTokenFactory = await ethers.getContractFactory("CredToken");
  const credToken = await CredTokenFactory.deploy(
    "MaaS Router CRED",
    "CRED",
    deployer.address
  );
  await credToken.waitForDeployment();
  
  const credTokenAddress = await credToken.getAddress();
  console.log("CredToken deployed to:", credTokenAddress);

  // Deploy Settlement
  console.log("\nDeploying Settlement...");
  const SettlementFactory = await ethers.getContractFactory("Settlement");
  const settlement = await SettlementFactory.deploy(
    credTokenAddress,
    deployer.address,
    0 // Settlement at 00:00 UTC
  );
  await settlement.waitForDeployment();
  
  const settlementAddress = await settlement.getAddress();
  console.log("Settlement deployed to:", settlementAddress);

  // Configure roles
  console.log("\nConfiguring roles...");
  
  // Grant SETTLER_ROLE to Settlement contract on CredToken
  await credToken.grantRole(await credToken.SETTLER_ROLE(), settlementAddress);
  console.log("Granted SETTLER_ROLE to Settlement contract");

  // Grant SETTLER_ROLE to deployer on Settlement
  await settlement.grantRole(await settlement.SETTLER_ROLE(), deployer.address);
  console.log("Granted SETTLER_ROLE to deployer on Settlement");

  // Verify deployment
  console.log("\n=== Deployment Summary ===");
  console.log("Network:", network.name);
  console.log("Chain ID:", network.chainId.toString());
  console.log("CredToken:", credTokenAddress);
  console.log("Settlement:", settlementAddress);
  console.log("Deployer:", deployer.address);

  // Save deployment info
  const deploymentInfo = {
    network: network.name,
    chainId: Number(network.chainId),
    credToken: credTokenAddress,
    settlement: settlementAddress,
    deployer: deployer.address,
    timestamp: new Date().toISOString(),
  };

  // Write to file
  const fs = require("fs");
  const deploymentsDir = "./deployments";
  if (!fs.existsSync(deploymentsDir)) {
    fs.mkdirSync(deploymentsDir);
  }
  
  const filename = `${deploymentsDir}/${network.name}-${Date.now()}.json`;
  fs.writeFileSync(filename, JSON.stringify(deploymentInfo, null, 2));
  console.log("\nDeployment info saved to:", filename);

  // Verification instructions
  console.log("\n=== Verification Commands ===");
  console.log(`npx hardhat verify --network ${network.name} ${credTokenAddress} "MaaS Router CRED" "CRED" ${deployer.address}`);
  console.log(`npx hardhat verify --network ${network.name} ${settlementAddress} ${credTokenAddress} ${deployer.address} 0`);
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });