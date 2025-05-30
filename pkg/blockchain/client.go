package blockchain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RPCClient defines the interface for blockchain RPC operations
type RPCClient interface {
	FetchBlockData(ctx context.Context, blockNumber uint64) (*BlockData, error)
	FetchTransactionReceipt(ctx context.Context, txHash string) (*TransactionReceipt, error)
	SetTimeout(timeout time.Duration)
}

// BaseRPCClient implements RPCClient for Base blockchain
type BaseRPCClient struct {
	url        string
	httpClient *http.Client
	timeout    time.Duration
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   *RPCError   `json:"error"`
	ID      int         `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// NewBaseRPCClient creates a new Base RPC client
func NewBaseRPCClient() RPCClient {
	return &BaseRPCClient{
		url: "https://mainnet.base.org/",
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		timeout: time.Second * 30,
	}
}

// NewBaseRPCClientWithURL creates a new Base RPC client with custom URL
func NewBaseRPCClientWithURL(url string) RPCClient {
	return &BaseRPCClient{
		url: url,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		timeout: time.Second * 30,
	}
}

// SetTimeout sets the timeout for RPC calls
func (c *BaseRPCClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.httpClient.Timeout = timeout
}

// callRPC makes a JSON-RPC call with exponential backoff retry logic
func (c *BaseRPCClient) callRPC(ctx context.Context, method string, params []interface{}) (interface{}, error) {
	const maxRetries = 12
	const baseDelay = time.Millisecond * 500
	const maxDelay = time.Second * 30

	request := RPCRequest{
		JsonRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("RPC call failed after %d attempts: %w", maxRetries, err)
			}

			// Exponential backoff with jitter
			delay := c.calculateBackoffDelay(attempt, baseDelay, maxDelay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to read response after %d attempts: %w", maxRetries, err)
			}

			delay := c.calculateBackoffDelay(attempt, baseDelay, maxDelay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		var rpcResp RPCResponse
		if err := json.Unmarshal(body, &rpcResp); err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("failed to unmarshal response after %d attempts: %w", maxRetries, err)
			}

			delay := c.calculateBackoffDelay(attempt, baseDelay, maxDelay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		if rpcResp.Error != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("RPC error after %d attempts: %s (code: %d)",
					maxRetries, rpcResp.Error.Message, rpcResp.Error.Code)
			}

			delay := c.calculateBackoffDelay(attempt, baseDelay, maxDelay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		return rpcResp.Result, nil
	}

	return nil, fmt.Errorf("unexpected error in RPC retry logic")
}

// calculateBackoffDelay calculates exponential backoff delay with jitter
func (c *BaseRPCClient) calculateBackoffDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add up to 25% jitter to avoid thundering herd
	jitter := time.Duration(rand.Float64() * 0.25 * float64(delay))
	return delay + jitter
}

// hexToUint64 converts hex string to uint64
func hexToUint64(hexStr string) (uint64, error) {
	if strings.HasPrefix(hexStr, "0x") {
		hexStr = hexStr[2:]
	}
	return strconv.ParseUint(hexStr, 16, 64)
}

// FetchTransactionReceipt fetches a transaction receipt
func (c *BaseRPCClient) FetchTransactionReceipt(ctx context.Context, txHash string) (*TransactionReceipt, error) {
	result, err := c.callRPC(ctx, "eth_getTransactionReceipt", []interface{}{txHash})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transaction receipt: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("transaction receipt not found for hash %s", txHash)
	}

	receiptData, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected receipt data format")
	}

	gasUsed, ok := receiptData["gasUsed"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid gasUsed in receipt")
	}

	status, ok := receiptData["status"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid status in receipt")
	}

	return &TransactionReceipt{
		TransactionHash: txHash,
		GasUsed:         gasUsed,
		Status:          status,
	}, nil
}

// FetchBlockData fetches a single block's data from Base with transaction receipts
func (c *BaseRPCClient) FetchBlockData(ctx context.Context, blockNumber uint64) (*BlockData, error) {
	blockHex := fmt.Sprintf("0x%x", blockNumber)

	// Get block with transaction details
	result, err := c.callRPC(ctx, "eth_getBlockByNumber", []interface{}{blockHex, true})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block %d: %w", blockNumber, err)
	}

	if result == nil {
		return nil, fmt.Errorf("block %d not found", blockNumber)
	}

	blockData, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected block data format for block %d", blockNumber)
	}

	// Parse block fields
	number, err := c.parseBlockNumber(blockData)
	if err != nil {
		return nil, fmt.Errorf("invalid block number in block %d: %w", blockNumber, err)
	}

	gasLimit, err := c.parseGasLimit(blockData)
	if err != nil {
		return nil, fmt.Errorf("invalid gas limit in block %d: %w", blockNumber, err)
	}

	gasUsed, err := c.parseGasUsed(blockData)
	if err != nil {
		return nil, fmt.Errorf("invalid gas used in block %d: %w", blockNumber, err)
	}

	baseFee, err := c.parseBaseFee(blockData)
	if err != nil {
		return nil, fmt.Errorf("invalid base fee in block %d: %w", blockNumber, err)
	}

	timestamp, err := c.parseTimestamp(blockData)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp in block %d: %w", blockNumber, err)
	}

	// Parse transactions
	transactions, err := c.parseTransactions(ctx, blockData, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transactions for block %d: %w", blockNumber, err)
	}

	return &BlockData{
		Number:        number,
		GasLimit:      gasLimit,
		GasUsed:       gasUsed,
		BaseFeePerGas: baseFee,
		Transactions:  transactions,
		Timestamp:     timestamp,
	}, nil
}

// Helper methods for parsing block data

func (c *BaseRPCClient) parseBlockNumber(blockData map[string]interface{}) (uint64, error) {
	numberStr, ok := blockData["number"].(string)
	if !ok {
		return 0, fmt.Errorf("missing or invalid number field")
	}
	return hexToUint64(numberStr)
}

func (c *BaseRPCClient) parseGasLimit(blockData map[string]interface{}) (uint64, error) {
	gasLimitStr, ok := blockData["gasLimit"].(string)
	if !ok {
		return 0, fmt.Errorf("missing or invalid gasLimit field")
	}
	return hexToUint64(gasLimitStr)
}

func (c *BaseRPCClient) parseGasUsed(blockData map[string]interface{}) (uint64, error) {
	gasUsedStr, ok := blockData["gasUsed"].(string)
	if !ok {
		return 0, fmt.Errorf("missing or invalid gasUsed field")
	}
	return hexToUint64(gasUsedStr)
}

func (c *BaseRPCClient) parseBaseFee(blockData map[string]interface{}) (uint64, error) {
	baseFeeHex, exists := blockData["baseFeePerGas"]
	if !exists || baseFeeHex == nil {
		return 0, nil // Pre-EIP-1559 blocks
	}

	baseFeeStr, ok := baseFeeHex.(string)
	if !ok {
		return 0, fmt.Errorf("invalid baseFeePerGas format")
	}
	return hexToUint64(baseFeeStr)
}

func (c *BaseRPCClient) parseTimestamp(blockData map[string]interface{}) (uint64, error) {
	timestampStr, ok := blockData["timestamp"].(string)
	if !ok {
		return 0, fmt.Errorf("missing or invalid timestamp field")
	}
	return hexToUint64(timestampStr)
}

func (c *BaseRPCClient) parseTransactions(ctx context.Context, blockData map[string]interface{}, blockNumber uint64) ([]Transaction, error) {
	txsData, exists := blockData["transactions"]
	if !exists {
		return []Transaction{}, nil
	}

	txsList, ok := txsData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected transactions format")
	}

	transactions := make([]Transaction, 0, len(txsList))

	for i, txData := range txsList {
		tx, ok := txData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected transaction format at index %d", i)
		}

		transaction, err := c.parseTransaction(ctx, tx)
		if err != nil {
			// Log warning but continue - don't fail entire block for one transaction
			fmt.Printf("Warning: failed to parse transaction in block %d: %v\n", blockNumber, err)
			continue
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (c *BaseRPCClient) parseTransaction(ctx context.Context, tx map[string]interface{}) (Transaction, error) {
	hash, ok := tx["hash"].(string)
	if !ok {
		return Transaction{}, fmt.Errorf("missing or invalid transaction hash")
	}

	gasStr, ok := tx["gas"].(string)
	if !ok {
		return Transaction{}, fmt.Errorf("missing or invalid gas field")
	}
	gas, err := hexToUint64(gasStr)
	if err != nil {
		return Transaction{}, fmt.Errorf("invalid gas value: %w", err)
	}

	txType, ok := tx["type"].(string)
	if !ok {
		txType = "0x0" // Default to legacy type
	}

	transaction := Transaction{
		Hash: hash,
		Gas:  gas,
		Type: txType,
	}

	// Parse optional fee fields
	if gasPrice, exists := tx["gasPrice"]; exists && gasPrice != nil {
		if gasPriceStr, ok := gasPrice.(string); ok {
			transaction.GasPrice, _ = hexToUint64(gasPriceStr)
		}
	}

	if maxFee, exists := tx["maxFeePerGas"]; exists && maxFee != nil {
		if maxFeeStr, ok := maxFee.(string); ok {
			transaction.MaxFeePerGas, _ = hexToUint64(maxFeeStr)
		}
	}

	if maxPriorityFee, exists := tx["maxPriorityFeePerGas"]; exists && maxPriorityFee != nil {
		if maxPriorityFeeStr, ok := maxPriorityFee.(string); ok {
			transaction.MaxPriorityFeePerGas, _ = hexToUint64(maxPriorityFeeStr)
		}
	}

	// Fetch receipt for actual gas used and status
	receipt, err := c.FetchTransactionReceipt(ctx, hash)
	if err != nil {
		// Use gas limit as fallback and assume success
		transaction.GasUsed = gas
		transaction.Status = 1
	} else {
		gasUsed, err := hexToUint64(receipt.GasUsed)
		if err != nil {
			gasUsed = gas // Fallback to gas limit
		}

		status, err := hexToUint64(receipt.Status)
		if err != nil {
			status = 1 // Assume success
		}

		transaction.GasUsed = gasUsed
		transaction.Status = status
	}

	return transaction, nil
}
