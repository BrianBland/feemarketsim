package blockchain

import "time"

// BlockData represents block data from Base blockchain
type BlockData struct {
	Number        uint64        `json:"number"`
	GasLimit      uint64        `json:"gasLimit"`
	GasUsed       uint64        `json:"gasUsed"`
	BaseFeePerGas uint64        `json:"baseFeePerGas"`
	Transactions  []Transaction `json:"transactions"`
	Timestamp     uint64        `json:"timestamp"`
}

// Transaction represents a transaction with relevant fee data
type Transaction struct {
	Hash                 string `json:"hash"`
	Gas                  uint64 `json:"gas"`     // Gas limit
	GasUsed              uint64 `json:"gasUsed"` // Actual gas used (from receipt)
	GasPrice             uint64 `json:"gasPrice,omitempty"`
	MaxFeePerGas         uint64 `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas uint64 `json:"maxPriorityFeePerGas,omitempty"`
	Type                 string `json:"type"`
	Status               uint64 `json:"status"` // Transaction status (1 = success, 0 = failed)
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash string `json:"transactionHash"`
	GasUsed         string `json:"gasUsed"`
	Status          string `json:"status"`
}

// DataSet represents a complete dataset from Base blockchain
type DataSet struct {
	StartBlock      uint64      `json:"startBlock"`
	EndBlock        uint64      `json:"endBlock"`
	InitialBaseFee  uint64      `json:"initialBaseFee"`
	InitialGasLimit uint64      `json:"initialGasLimit"`
	Blocks          []BlockData `json:"blocks"`
	FetchedAt       int64       `json:"fetchedAt"`
}

// SimulationResult represents the result of simulating against real Base data
type SimulationResult struct {
	TotalTransactions    int     `json:"totalTransactions"`
	DroppedTransactions  int     `json:"droppedTransactions"`
	DroppedPercentage    float64 `json:"droppedPercentage"`
	AvgBaseFee           uint64  `json:"avgBaseFee"`
	MaxBaseFee           uint64  `json:"maxBaseFee"`
	MinBaseFee           uint64  `json:"minBaseFee"`
	TotalGasUsed         uint64  `json:"totalGasUsed"`
	EffectiveUtilization float64 `json:"effectiveUtilization"`
	// Extended data for visualization
	ComparisonData *ComparisonData `json:"comparisonData,omitempty"`
}

// ComparisonData holds detailed simulation data for visualization
type ComparisonData struct {
	BlockNumbers       []float64 `json:"blockNumbers"`
	ActualBaseFees     []float64 `json:"actualBaseFees"`
	SimulatedBaseFees  []float64 `json:"simulatedBaseFees"`
	DroppedPercentages []float64 `json:"droppedPercentages"`
	ActualGasUsages    []float64 `json:"actualGasUsages"`
	EffectiveGasUsages []float64 `json:"effectiveGasUsages"`
	LearningRates      []float64 `json:"learningRates"`
}

// FetchProgress represents progress information during block fetching
type FetchProgress struct {
	Total     int
	Completed int
	Failed    int
	Round     int
	StartTime time.Time
}

// FetchOptions contains options for blockchain data fetching
type FetchOptions struct {
	StartBlock uint64
	EndBlock   uint64
	Workers    int
	MaxRetries int
	Timeout    time.Duration
}

// DefaultFetchOptions returns sensible defaults for fetching
func DefaultFetchOptions(startBlock, endBlock uint64) FetchOptions {
	return FetchOptions{
		StartBlock: startBlock,
		EndBlock:   endBlock,
		Workers:    64,
		MaxRetries: 5,
		Timeout:    time.Second * 30,
	}
}
