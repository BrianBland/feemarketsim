package blockchain

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockRPCClient implements RPCClient interface for testing
type MockRPCClient struct {
	blocks map[uint64]*BlockData
	delay  time.Duration
	errors map[uint64]error
}

// NewMockRPCClient creates a new mock RPC client
func NewMockRPCClient() *MockRPCClient {
	return &MockRPCClient{
		blocks: make(map[uint64]*BlockData),
		errors: make(map[uint64]error),
	}
}

// AddMockBlock adds a mock block to the client
func (m *MockRPCClient) AddMockBlock(block *BlockData) {
	m.blocks[block.Number] = block
}

// SetError sets an error for a specific block number
func (m *MockRPCClient) SetError(blockNumber uint64, err error) {
	m.errors[blockNumber] = err
}

// SetDelay sets a delay for all requests (to simulate network latency)
func (m *MockRPCClient) SetDelay(delay time.Duration) {
	m.delay = delay
}

// FetchBlockData implements the RPCClient interface
func (m *MockRPCClient) FetchBlockData(ctx context.Context, blockNumber uint64) (*BlockData, error) {
	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}

	if err, exists := m.errors[blockNumber]; exists {
		return nil, err
	}

	if block, exists := m.blocks[blockNumber]; exists {
		return block, nil
	}

	return nil, &RPCError{Code: -1, Message: "block not found"}
}

// FetchTransactionReceipt implements the RPCClient interface (not used in tests currently)
func (m *MockRPCClient) FetchTransactionReceipt(ctx context.Context, txHash string) (*TransactionReceipt, error) {
	return &TransactionReceipt{
		TransactionHash: txHash,
		GasUsed:         "0x5208", // 21000 gas
		Status:          "0x1",    // Success
	}, nil
}

// SetTimeout implements the RPCClient interface (no-op for mock)
func (m *MockRPCClient) SetTimeout(timeout time.Duration) {
	// No-op for mock
}

func TestHexToUint64(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
		hasError bool
	}{
		{"0x0", 0, false},
		{"0x1", 1, false},
		{"0x10", 16, false},
		{"0xFF", 255, false},
		{"0x1234", 4660, false},
		{"10", 16, false}, // Without 0x prefix
		{"FF", 255, false},
		{"invalid", 0, true},
		{"0xGG", 0, true},
	}

	for _, test := range tests {
		result, err := hexToUint64(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("For input %s, expected %d but got %d", test.input, test.expected, result)
			}
		}
	}
}

func TestBlockFetcher_SingleBlock(t *testing.T) {
	mockClient := NewMockRPCClient()

	// Add a mock block
	mockBlock := &BlockData{
		Number:        100,
		GasLimit:      30000000,
		GasUsed:       15000000,
		BaseFeePerGas: 1000000000,
		Transactions:  []Transaction{},
		Timestamp:     1234567890,
	}
	mockClient.AddMockBlock(mockBlock)

	options := FetchOptions{
		StartBlock: 100,
		EndBlock:   100,
		Workers:    1,
		MaxRetries: 3,
		Timeout:    time.Second * 5,
	}

	fetcher := NewBlockFetcher(mockClient, options)
	ctx := context.Background()

	dataset, err := fetcher.FetchRange(ctx, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(dataset.Blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(dataset.Blocks))
	}

	if dataset.Blocks[0].Number != 100 {
		t.Errorf("Expected block number 100, got %d", dataset.Blocks[0].Number)
	}

	if dataset.InitialBaseFee != mockBlock.BaseFeePerGas {
		t.Errorf("Expected initial base fee %d, got %d", mockBlock.BaseFeePerGas, dataset.InitialBaseFee)
	}
}

func TestBlockFetcher_MultipleBlocks(t *testing.T) {
	mockClient := NewMockRPCClient()

	// Add multiple mock blocks
	for i := uint64(100); i <= 105; i++ {
		mockBlock := &BlockData{
			Number:        i,
			GasLimit:      30000000,
			GasUsed:       15000000 + i*1000000, // Varying gas usage
			BaseFeePerGas: 1000000000 + i*100000000,
			Transactions:  []Transaction{},
			Timestamp:     1234567890 + i,
		}
		mockClient.AddMockBlock(mockBlock)
	}

	options := FetchOptions{
		StartBlock: 100,
		EndBlock:   105,
		Workers:    2,
		MaxRetries: 3,
		Timeout:    time.Second * 5,
	}

	fetcher := NewBlockFetcher(mockClient, options)
	ctx := context.Background()

	dataset, err := fetcher.FetchRange(ctx, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedBlocks := 6
	if len(dataset.Blocks) != expectedBlocks {
		t.Fatalf("Expected %d blocks, got %d", expectedBlocks, len(dataset.Blocks))
	}

	// Verify blocks are in order
	for i, block := range dataset.Blocks {
		expectedNumber := uint64(100 + i)
		if block.Number != expectedNumber {
			t.Errorf("Block at index %d: expected number %d, got %d", i, expectedNumber, block.Number)
		}
	}
}

func TestBlockFetcher_WithErrors(t *testing.T) {
	mockClient := NewMockRPCClient()

	// Add some blocks but not all
	for i := uint64(100); i <= 102; i++ {
		mockBlock := &BlockData{
			Number:        i,
			GasLimit:      30000000,
			GasUsed:       15000000,
			BaseFeePerGas: 1000000000,
			Transactions:  []Transaction{},
			Timestamp:     1234567890,
		}
		mockClient.AddMockBlock(mockBlock)
	}

	// Block 103 will fail (not added)
	// Block 104 will succeed
	mockBlock104 := &BlockData{
		Number:        104,
		GasLimit:      30000000,
		GasUsed:       15000000,
		BaseFeePerGas: 1000000000,
		Transactions:  []Transaction{},
		Timestamp:     1234567890,
	}
	mockClient.AddMockBlock(mockBlock104)

	options := FetchOptions{
		StartBlock: 100,
		EndBlock:   104,
		Workers:    2,
		MaxRetries: 2, // Limited retries to make test faster
		Timeout:    time.Second * 5,
	}

	fetcher := NewBlockFetcher(mockClient, options)
	ctx := context.Background()

	// This should fail because block 103 is missing
	_, err := fetcher.FetchRange(ctx, nil)
	if err == nil {
		t.Fatalf("Expected error due to missing block, but got none")
	}

	if !strings.Contains(err.Error(), "unable to fetch complete dataset") {
		t.Errorf("Expected error about unable to fetch complete dataset, got: %v", err)
	}
}

func TestValidateDataSet(t *testing.T) {
	tests := []struct {
		name      string
		dataset   *DataSet
		expectErr bool
	}{
		{
			name:      "nil dataset",
			dataset:   nil,
			expectErr: true,
		},
		{
			name: "empty blocks",
			dataset: &DataSet{
				StartBlock: 100,
				EndBlock:   100,
				Blocks:     []BlockData{},
			},
			expectErr: true,
		},
		{
			name: "block count mismatch",
			dataset: &DataSet{
				StartBlock: 100,
				EndBlock:   102,
				Blocks: []BlockData{
					{Number: 100},
					// Missing block 101 and 102
				},
			},
			expectErr: true,
		},
		{
			name: "block number gap",
			dataset: &DataSet{
				StartBlock: 100,
				EndBlock:   102,
				Blocks: []BlockData{
					{Number: 100},
					{Number: 102}, // Missing 101
					{Number: 103}, // Should be 102
				},
			},
			expectErr: true,
		},
		{
			name: "valid dataset",
			dataset: &DataSet{
				StartBlock: 100,
				EndBlock:   102,
				Blocks: []BlockData{
					{Number: 100},
					{Number: 101},
					{Number: 102},
				},
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateDataSet(test.dataset)

			if test.expectErr && err == nil {
				t.Errorf("Expected error for test %s, but got none", test.name)
			}

			if !test.expectErr && err != nil {
				t.Errorf("Unexpected error for test %s: %v", test.name, err)
			}
		})
	}
}

func TestDefaultFetchOptions(t *testing.T) {
	startBlock := uint64(100)
	endBlock := uint64(200)

	options := DefaultFetchOptions(startBlock, endBlock)

	if options.StartBlock != startBlock {
		t.Errorf("Expected StartBlock %d, got %d", startBlock, options.StartBlock)
	}

	if options.EndBlock != endBlock {
		t.Errorf("Expected EndBlock %d, got %d", endBlock, options.EndBlock)
	}

	if options.Workers <= 0 {
		t.Errorf("Expected positive Workers, got %d", options.Workers)
	}

	if options.MaxRetries <= 0 {
		t.Errorf("Expected positive MaxRetries, got %d", options.MaxRetries)
	}

	if options.Timeout <= 0 {
		t.Errorf("Expected positive Timeout, got %v", options.Timeout)
	}
}
