// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "@openzeppelin/contracts/access/AccessControl.sol";
import "@openzeppelin/contracts/security/Pausable.sol";

/**
 * @title CredToken
 * @dev $CRED Token for MaaS Router - AI API Gateway
 * 
 * Features:
 * - ERC20 standard with burn capability
 * - Role-based access control
 * - Pausable transfers
 * - Minting controlled by MINTER_ROLE
 * - Settlement integration
 */
contract CredToken is ERC20, ERC20Burnable, AccessControl, Pausable {
    bytes32 public constant MINTER_ROLE = keccak256("MINTER_ROLE");
    bytes32 public constant PAUSER_ROLE = keccak256("PAUSER_ROLE");
    bytes32 public constant SETTLER_ROLE = keccak256("SETTLER_ROLE");

    // Settlement contract address
    address public settlementContract;
    
    // Maximum supply cap
    uint256 public constant MAX_SUPPLY = 1_000_000_000 * 10**18; // 1 billion tokens
    
    // Events
    event SettlementContractUpdated(address indexed oldContract, address indexed newContract);
    event TokensMinted(address indexed to, uint256 amount, string reason);
    event SettlementProcessed(bytes32 indexed merkleRoot, uint256 totalAmount, uint256 timestamp);

    constructor(
        string memory name,
        string memory symbol,
        address admin
    ) ERC20(name, symbol) {
        require(admin != address(0), "Invalid admin address");
        
        _grantRole(DEFAULT_ADMIN_ROLE, admin);
        _grantRole(MINTER_ROLE, admin);
        _grantRole(PAUSER_ROLE, admin);
        _grantRole(SETTLER_ROLE, admin);
    }

    /**
     * @dev Mint new tokens (only MINTER_ROLE)
     * @param to Recipient address
     * @param amount Amount to mint
     * @param reason Reason for minting
     */
    function mint(
        address to,
        uint256 amount,
        string calldata reason
    ) external onlyRole(MINTER_ROLE) {
        require(totalSupply() + amount <= MAX_SUPPLY, "Max supply exceeded");
        require(to != address(0), "Invalid recipient");
        
        _mint(to, amount);
        emit TokensMinted(to, amount, reason);
    }

    /**
     * @dev Batch mint tokens to multiple recipients
     * @param recipients Array of recipient addresses
     * @param amounts Array of amounts
     */
    function batchMint(
        address[] calldata recipients,
        uint256[] calldata amounts,
        string calldata reason
    ) external onlyRole(MINTER_ROLE) {
        require(recipients.length == amounts.length, "Length mismatch");
        
        uint256 totalAmount = 0;
        for (uint256 i = 0; i < amounts.length; i++) {
            totalAmount += amounts[i];
        }
        
        require(totalSupply() + totalAmount <= MAX_SUPPLY, "Max supply exceeded");
        
        for (uint256 i = 0; i < recipients.length; i++) {
            require(recipients[i] != address(0), "Invalid recipient");
            _mint(recipients[i], amounts[i]);
        }
        
        emit TokensMinted(address(0), totalAmount, reason);
    }

    /**
     * @dev Process daily settlement (only SETTLER_ROLE)
     * @param merkleRoot Merkle root of settlement data
     * @param totalAmount Total amount to settle
     * @param recipients Array of recipient addresses
     * @param amounts Array of settlement amounts
     */
    function processSettlement(
        bytes32 merkleRoot,
        uint256 totalAmount,
        address[] calldata recipients,
        uint256[] calldata amounts
    ) external onlyRole(SETTLER_ROLE) whenNotPaused {
        require(recipients.length == amounts.length, "Length mismatch");
        require(totalSupply() + totalAmount <= MAX_SUPPLY, "Max supply exceeded");
        
        for (uint256 i = 0; i < recipients.length; i++) {
            require(recipients[i] != address(0), "Invalid recipient");
            _mint(recipients[i], amounts[i]);
        }
        
        emit SettlementProcessed(merkleRoot, totalAmount, block.timestamp);
    }

    /**
     * @dev Set settlement contract address
     * @param _settlementContract New settlement contract address
     */
    function setSettlementContract(address _settlementContract) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(_settlementContract != address(0), "Invalid address");
        address oldContract = settlementContract;
        settlementContract = _settlementContract;
        emit SettlementContractUpdated(oldContract, _settlementContract);
    }

    /**
     * @dev Pause token transfers
     */
    function pause() external onlyRole(PAUSER_ROLE) {
        _pause();
    }

    /**
     * @dev Unpause token transfers
     */
    function unpause() external onlyRole(PAUSER_ROLE) {
        _unpause();
    }

    /**
     * @dev Override _beforeTokenTransfer to include pausable logic
     */
    function _beforeTokenTransfer(
        address from,
        address to,
        uint256 amount
    ) internal override whenNotPaused {
        super._beforeTokenTransfer(from, to, amount);
    }

    /**
     * @dev Get token info
     */
    function getTokenInfo() external view returns (
        string memory name,
        string memory symbol,
        uint256 totalSupply,
        uint256 maxSupply,
        uint8 decimals
    ) {
        return (
            name(),
            symbol(),
            totalSupply(),
            MAX_SUPPLY,
            decimals()
        );
    }

    /**
     * @dev Get user balance with detailed info
     */
    function getBalanceInfo(address account) external view returns (
        uint256 balance,
        uint256 lockedAmount
    ) {
        return (balanceOf(account), 0);
    }
}