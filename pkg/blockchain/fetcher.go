package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// BlockFetcher handles concurrent fetching of blockchain data
type BlockFetcher struct {
	client  RPCClient
	options FetchOptions
}

// BlockFetchJob represents a block fetching job for the worker pool
type BlockFetchJob struct {
	BlockNumber uint64
	Result      chan *BlockFetchResult
}

// BlockFetchResult represents the result of fetching a block
type BlockFetchResult struct {
	Block *BlockData
	Error error
}

// ProgressCallback is called to report progress during fetching
type ProgressCallback func(progress FetchProgress)

// NewBlockFetcher creates a new block fetcher
func NewBlockFetcher(client RPCClient, options FetchOptions) *BlockFetcher {
	return &BlockFetcher{
		client:  client,
		options: options,
	}
}

// FetchRange fetches a range of blocks with concurrency and ensures no gaps
func (f *BlockFetcher) FetchRange(ctx context.Context, progressCallback ProgressCallback) (*DataSet, error) {
	fmt.Printf("Fetching Base blockchain data from block %d to %d (%d blocks)...\n",
		f.options.StartBlock, f.options.EndBlock, f.options.EndBlock-f.options.StartBlock+1)

	totalBlocks := f.options.EndBlock - f.options.StartBlock + 1

	// Track which blocks we need to fetch
	remainingBlocks := make(map[uint64]bool)
	for blockNum := f.options.StartBlock; blockNum <= f.options.EndBlock; blockNum++ {
		remainingBlocks[blockNum] = true
	}

	results := make(map[uint64]*BlockFetchResult)
	var firstBlock *BlockData

	progress := FetchProgress{
		Total:     int(totalBlocks),
		StartTime: time.Now(),
	}

	// Retry logic with multiple rounds
	for round := 1; round <= f.options.MaxRetries && len(remainingBlocks) > 0; round++ {
		fmt.Printf("\n=== Fetch Round %d: %d blocks remaining ===\n", round, len(remainingBlocks))

		progress.Round = round
		progress.Failed = len(remainingBlocks)
		if progressCallback != nil {
			progressCallback(progress)
		}

		roundResults, err := f.fetchRound(ctx, remainingBlocks, round)
		if err != nil {
			return nil, fmt.Errorf("failed in round %d: %w", round, err)
		}

		// Process round results
		for blockNum, result := range roundResults {
			if result.Error != nil {
				fmt.Printf("Round %d: Block %d failed: %v\n", round, blockNum, result.Error)
				// Keep this block in remainingBlocks for next round
			} else {
				// Successfully fetched block
				results[blockNum] = result
				delete(remainingBlocks, blockNum)

				if blockNum == f.options.StartBlock {
					firstBlock = result.Block
				}

				progress.Completed++
				progress.Failed = len(remainingBlocks)
			}
		}

		if progressCallback != nil {
			progressCallback(progress)
		}

		if len(remainingBlocks) == 0 {
			fmt.Printf("✅ All blocks successfully fetched in %d rounds!\n", round)
			break
		} else if round < f.options.MaxRetries {
			fmt.Printf("⚠️  %d blocks still missing, will retry in round %d\n", len(remainingBlocks), round+1)
			// Brief pause before next round
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second * 2):
			}
		} else {
			return nil, f.handleMissingBlocks(remainingBlocks)
		}
	}

	if firstBlock == nil {
		return nil, fmt.Errorf("failed to fetch first block %d", f.options.StartBlock)
	}

	// Validate no gaps and create dataset
	return f.createDataSet(results, firstBlock)
}

// fetchRound executes one round of concurrent block fetching
func (f *BlockFetcher) fetchRound(ctx context.Context, remainingBlocks map[uint64]bool, round int) (map[uint64]*BlockFetchResult, error) {
	jobs := make(chan BlockFetchJob, len(remainingBlocks))
	resultChan := make(chan *BlockFetchResult, len(remainingBlocks))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < f.options.Workers; i++ {
		wg.Add(1)
		go f.worker(ctx, jobs, &wg)
	}

	// Send jobs for remaining blocks
	go func() {
		defer close(jobs)
		for blockNum := range remainingBlocks {
			select {
			case jobs <- BlockFetchJob{
				BlockNumber: blockNum,
				Result:      resultChan,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results for this round
	roundResults := make(map[uint64]*BlockFetchResult)
	completed := 0
	roundStartTime := time.Now()

	for result := range resultChan {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		completed++

		if result.Block != nil {
			roundResults[result.Block.Number] = result
		} else if result.Error != nil {
			// We need to track which block this error belongs to
			// This is a limitation of our current design - we should improve this
			fmt.Printf("Round %d: Received error without block number: %v\n", round, result.Error)
		}

		// Progress reporting for this round
		if completed%50 == 0 || completed == len(remainingBlocks) {
			elapsed := time.Since(roundStartTime)
			fmt.Printf("Round %d progress: %d/%d completed in %v\n",
				round, completed, len(remainingBlocks), elapsed)
		}
	}

	return roundResults, nil
}

// worker function for concurrent block fetching
func (f *BlockFetcher) worker(ctx context.Context, jobs <-chan BlockFetchJob, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			job.Result <- &BlockFetchResult{
				Block: nil,
				Error: ctx.Err(),
			}
			return
		default:
		}

		block, err := f.client.FetchBlockData(ctx, job.BlockNumber)
		job.Result <- &BlockFetchResult{
			Block: block,
			Error: err,
		}
	}
}

// handleMissingBlocks handles the case where some blocks couldn't be fetched
func (f *BlockFetcher) handleMissingBlocks(remainingBlocks map[uint64]bool) error {
	fmt.Printf("❌ Failed to fetch %d blocks after %d rounds\n", len(remainingBlocks), f.options.MaxRetries)

	// List the specific missing blocks
	var missingBlocks []uint64
	for blockNum := range remainingBlocks {
		missingBlocks = append(missingBlocks, blockNum)
	}

	if len(missingBlocks) <= 20 {
		fmt.Printf("Missing blocks: %v\n", missingBlocks)
	} else {
		fmt.Printf("Missing blocks: %v... (and %d more)\n", missingBlocks[:20], len(missingBlocks)-20)
	}

	return fmt.Errorf("unable to fetch complete dataset: %d blocks missing after %d retry rounds",
		len(remainingBlocks), f.options.MaxRetries)
}

// createDataSet creates a validated dataset from fetched blocks
func (f *BlockFetcher) createDataSet(results map[uint64]*BlockFetchResult, firstBlock *BlockData) (*DataSet, error) {
	// Create ordered slice of blocks - should have no gaps now
	var blocks []BlockData
	successCount := 0
	var missingBlocks []uint64

	for blockNum := f.options.StartBlock; blockNum <= f.options.EndBlock; blockNum++ {
		if result, exists := results[blockNum]; exists && result.Error == nil {
			blocks = append(blocks, *result.Block)
			successCount++
		} else {
			missingBlocks = append(missingBlocks, blockNum)
		}
	}

	totalBlocks := int(f.options.EndBlock - f.options.StartBlock + 1)
	fmt.Printf("\n=== Final Results ===\n")
	fmt.Printf("Successfully fetched: %d out of %d blocks (%.2f%%)\n",
		successCount, totalBlocks, float64(successCount)/float64(totalBlocks)*100)

	if len(missingBlocks) > 0 {
		fmt.Printf("❌ GAPS DETECTED: %d missing blocks\n", len(missingBlocks))
		if len(missingBlocks) <= 10 {
			fmt.Printf("Missing blocks: %v\n", missingBlocks)
		} else {
			fmt.Printf("Missing blocks: %v... (and %d more)\n", missingBlocks[:10], len(missingBlocks)-10)
		}
		return nil, fmt.Errorf("incomplete dataset: %d blocks missing, cannot proceed with gaps", len(missingBlocks))
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("no blocks were successfully fetched")
	}

	// Create dataset
	dataset := &DataSet{
		StartBlock:      f.options.StartBlock,
		EndBlock:        f.options.EndBlock,
		InitialBaseFee:  firstBlock.BaseFeePerGas,
		InitialGasLimit: firstBlock.GasLimit,
		Blocks:          blocks,
		FetchedAt:       time.Now().Unix(),
	}

	return dataset, nil
}

// SaveToFile saves a dataset to a JSON file
func SaveDataSetToFile(dataset *DataSet, filename string) error {
	jsonData, err := json.MarshalIndent(dataset, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("✅ Complete dataset with NO GAPS saved to %s\n", filename)
	return nil
}

// LoadDataSetFromFile loads a dataset from a JSON file
func LoadDataSetFromFile(filename string) (*DataSet, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var dataset DataSet
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dataset: %w", err)
	}

	return &dataset, nil
}

// ValidateDataSet performs validation checks on a dataset
func ValidateDataSet(dataset *DataSet) error {
	if dataset == nil {
		return fmt.Errorf("dataset is nil")
	}

	if len(dataset.Blocks) == 0 {
		return fmt.Errorf("dataset contains no blocks")
	}

	expectedBlocks := int(dataset.EndBlock - dataset.StartBlock + 1)
	if len(dataset.Blocks) != expectedBlocks {
		return fmt.Errorf("dataset block count mismatch: expected %d, got %d",
			expectedBlocks, len(dataset.Blocks))
	}

	// Check for gaps in block numbers
	for i, block := range dataset.Blocks {
		expectedBlockNum := dataset.StartBlock + uint64(i)
		if block.Number != expectedBlockNum {
			return fmt.Errorf("block number gap detected: expected %d, got %d at index %d",
				expectedBlockNum, block.Number, i)
		}
	}

	return nil
}
