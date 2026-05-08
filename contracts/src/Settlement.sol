// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/AccessControl.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/utils/cryptography/MerkleProof.sol";

interface ICredToken {
    function processSettlement(
        bytes32 merkleRoot,
        uint256 totalAmount,
        address[] calldata recipients,
        uint256[] calldata amounts
    ) external;
}

/**
 * @title Settlement
 * @dev Daily settlement contract for MaaS Router billing
 * 
 * Features:
 * - Merkle tree based settlement verification
 * - Daily settlement scheduling
 * - Emergency pause functionality
 * - Settlement history tracking
 */
contract Settlement is AccessControl, ReentrancyGuard {
    bytes32 public constant SETTLER_ROLE = keccak256("SETTLER_ROLE");
    bytes32 public constant OPERATOR_ROLE = keccak256("OPERATOR_ROLE");

    ICredToken public credToken;
    
    // Settlement period (24 hours)
    uint256 public constant SETTLEMENT_PERIOD = 1 days;
    
    // Last settlement timestamp
    uint256 public lastSettlementTime;
    
    // Settlement hour in UTC (0-23)
    uint256 public settlementHour;
    
    // Settlement records
    struct SettlementRecord {
        bytes32 merkleRoot;
        uint256 totalAmount;
        uint256 timestamp;
        uint256 userCount;
        bool processed;
    }
    
    // Mapping: date => settlement record
    mapping(uint256 => SettlementRecord) public settlements;
    
    // Mapping: merkleRoot => bool (prevent double settlement)
    mapping(bytes32 => bool) public processedRoots;
    
    // Mapping: user => last settlement date
    mapping(address => uint256) public lastUserSettlement;
    
    // Events
    event SettlementScheduled(uint256 indexed date, uint256 scheduledTime);
    event SettlementProcessed(
        uint256 indexed date,
        bytes32 indexed merkleRoot,
        uint256 totalAmount,
        uint256 userCount
    );
    event SettlementHourUpdated(uint256 oldHour, uint256 newHour);
    event CredTokenUpdated(address indexed oldToken, address indexed newToken);
    event EmergencyWithdraw(address indexed token, uint256 amount);

    modifier onlySettler() {
        require(hasRole(SETTLER_ROLE, msg.sender), "Not settler");
        _;
    }

    constructor(
        address _credToken,
        address admin,
        uint256 _settlementHour
    ) {
        require(_credToken != address(0), "Invalid token address");
        require(admin != address(0), "Invalid admin address");
        require(_settlementHour < 24, "Invalid settlement hour");
        
        credToken = ICredToken(_credToken);
        settlementHour = _settlementHour;
        lastSettlementTime = block.timestamp;
        
        _grantRole(DEFAULT_ADMIN_ROLE, admin);
        _grantRole(SETTLER_ROLE, admin);
        _grantRole(OPERATOR_ROLE, admin);
    }

    /**
     * @dev Submit settlement data (only SETTLER_ROLE)
     * @param date Settlement date (timestamp of day start)
     * @param merkleRoot Merkle root of settlement data
     * @param totalAmount Total CRED amount to distribute
     * @param userCount Number of users in settlement
     * @param recipients Array of recipient addresses
     * @param amounts Array of amounts for each recipient
     * @param merkleProofs Array of merkle proofs
     */
    function submitSettlement(
        uint256 date,
        bytes32 merkleRoot,
        uint256 totalAmount,
        uint256 userCount,
        address[] calldata recipients,
        uint256[] calldata amounts,
        bytes32[][] calldata merkleProofs
    ) external onlySettler nonReentrant {
        require(!processedRoots[merkleRoot], "Root already processed");
        require(recipients.length == amounts.length, "Length mismatch");
        require(recipients.length == merkleProofs.length, "Proof length mismatch");
        require(date < block.timestamp, "Future date");
        require(date >= lastSettlementTime - SETTLEMENT_PERIOD, "Date too old");
        
        // Verify merkle proofs for each recipient
        for (uint256 i = 0; i < recipients.length; i++) {
            bytes32 leaf = keccak256(abi.encodePacked(recipients[i], amounts[i]));
            require(
                MerkleProof.verify(merkleProofs[i], merkleRoot, leaf),
                "Invalid merkle proof"
            );
        }
        
        // Process settlement through CredToken
        credToken.processSettlement(merkleRoot, totalAmount, recipients, amounts);
        
        // Record settlement
        settlements[date] = SettlementRecord({
            merkleRoot: merkleRoot,
            totalAmount: totalAmount,
            timestamp: block.timestamp,
            userCount: userCount,
            processed: true
        });
        
        processedRoots[merkleRoot] = true;
        lastSettlementTime = block.timestamp;
        
        // Update last settlement time for users
        for (uint256 i = 0; i < recipients.length; i++) {
            lastUserSettlement[recipients[i]] = date;
        }
        
        emit SettlementProcessed(date, merkleRoot, totalAmount, userCount);
    }

    /**
     * @dev Submit settlement with simplified interface (trusted mode)
     * @param date Settlement date
     * @param merkleRoot Merkle root of settlement data
     * @param totalAmount Total CRED amount
     * @param userCount Number of users
     * @param recipients Recipient addresses
     * @param amounts Settlement amounts
     */
    function submitSettlementSimple(
        uint256 date,
        bytes32 merkleRoot,
        uint256 totalAmount,
        uint256 userCount,
        address[] calldata recipients,
        uint256[] calldata amounts
    ) external onlySettler nonReentrant {
        require(!processedRoots[merkleRoot], "Root already processed");
        require(recipients.length == amounts.length, "Length mismatch");
        require(date < block.timestamp, "Future date");
        
        // Process settlement
        credToken.processSettlement(merkleRoot, totalAmount, recipients, amounts);
        
        settlements[date] = SettlementRecord({
            merkleRoot: merkleRoot,
            totalAmount: totalAmount,
            timestamp: block.timestamp,
            userCount: userCount,
            processed: true
        });
        
        processedRoots[merkleRoot] = true;
        lastSettlementTime = block.timestamp;
        
        for (uint256 i = 0; i < recipients.length; i++) {
            lastUserSettlement[recipients[i]] = date;
        }
        
        emit SettlementProcessed(date, merkleRoot, totalAmount, userCount);
    }

    /**
     * @dev Get next settlement time
     */
    function getNextSettlementTime() public view returns (uint256) {
        uint256 currentDay = block.timestamp / 1 days;
        uint256 nextSettlement = (currentDay * 1 days) + (settlementHour * 1 hours);
        
        if (nextSettlement <= block.timestamp) {
            nextSettlement += 1 days;
        }
        
        return nextSettlement;
    }

    /**
     * @dev Check if settlement is due
     */
    function isSettlementDue() external view returns (bool) {
        return block.timestamp >= getNextSettlementTime();
    }

    /**
     * @dev Get settlement info for a specific date
     */
    function getSettlementInfo(uint256 date) external view returns (SettlementRecord memory) {
        return settlements[date];
    }

    /**
     * @dev Get user's last settlement date
     */
    function getUserLastSettlement(address user) external view returns (uint256) {
        return lastUserSettlement[user];
    }

    /**
     * @dev Update settlement hour (only admin)
     * @param newHour New settlement hour (0-23 UTC)
     */
    function setSettlementHour(uint256 newHour) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(newHour < 24, "Invalid hour");
        uint256 oldHour = settlementHour;
        settlementHour = newHour;
        emit SettlementHourUpdated(oldHour, newHour);
    }

    /**
     * @dev Update CRED token address (only admin)
     * @param newToken New token address
     */
    function setCredToken(address newToken) external onlyRole(DEFAULT_ADMIN_ROLE) {
        require(newToken != address(0), "Invalid address");
        address oldToken = address(credToken);
        credToken = ICredToken(newToken);
        emit CredTokenUpdated(oldToken, newToken);
    }

    /**
     * @dev Get settlement statistics
     */
    function getSettlementStats() external view returns (
        uint256 totalSettlements,
        uint256 lastSettlement,
        uint256 nextSettlement,
        bool isDue
    ) {
        uint256 currentDay = block.timestamp / 1 days;
        uint256 count = 0;
        
        for (uint256 i = 0; i < 30; i++) {
            if (settlements[currentDay - i].processed) {
                count++;
            }
        }
        
        return (
            count,
            lastSettlementTime,
            getNextSettlementTime(),
            block.timestamp >= getNextSettlementTime()
        );
    }
}