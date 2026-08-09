package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	// StateRentDenom is the token used to pay state rent.
	StateRentDenom = "uzytc"

	// DefaultRentRatePerBytePerBlock is 0.000001 ZYTC per byte per block.
	// 21KB ciphertext × 1 rate × 14400 blocks/day = 302,400 uzytc/day = 0.3024 ZYTC/day.
	// At $0.10/ZYTC, that's ~$0.03/day to store an FheUint32 ciphertext.
	DefaultRentRatePerBytePerBlock int64 = 1 // in uzytc (micro-ZYTC)

	// DefaultMaxFreeSizeBytes: first 1KB of storage per address is free.
	// This covers small metadata, Dilithium5 pubkey registrations, etc.
	DefaultMaxFreeSizeBytes int64 = 1024

	// DefaultGracePeriodBlocks: 14400 blocks ≈ 1 day before eviction.
	// Users have 1 day after balance hits zero to top up before data is pruned.
	DefaultGracePeriodBlocks int64 = 14400
)

// StateRentParams controls the economics of on-chain encrypted storage.
// All fields are governance-adjustable via parameter change proposals.
type StateRentParams struct {
	// RentRatePerBytePerBlock is the rent in uzytc per byte per block.
	// Governance can adjust this to respond to actual storage costs.
	RentRatePerBytePerBlock int64 `json:"rent_rate_per_byte_per_block"`

	// MaxFreeSizeBytes is the per-address free storage quota in bytes.
	// Storage below this threshold is never charged rent.
	MaxFreeSizeBytes int64 `json:"max_free_size_bytes"`

	// GracePeriodBlocks is the number of blocks a commitment survives
	// after the rent balance is exhausted, before being pruned.
	GracePeriodBlocks int64 `json:"grace_period_blocks"`
}

// DefaultStateRentParams returns conservative bootstrap parameters suitable
// for testnet launch. Validators and token holders adjust these via governance.
func DefaultStateRentParams() StateRentParams {
	return StateRentParams{
		RentRatePerBytePerBlock: DefaultRentRatePerBytePerBlock,
		MaxFreeSizeBytes:        DefaultMaxFreeSizeBytes,
		GracePeriodBlocks:       DefaultGracePeriodBlocks,
	}
}

// RentDue calculates the rent owed for a given storage size and number of blocks.
// Returns a zero coin if sizeBytes <= MaxFreeSizeBytes (free tier).
//
// Formula: max(0, sizeBytes - MaxFreeSize) * RentRatePerBytePerBlock * blocks
func (p StateRentParams) RentDue(sizeBytes, blocks int64) sdk.Coin {
	billableBytes := sizeBytes - p.MaxFreeSizeBytes
	if billableBytes <= 0 || blocks <= 0 {
		return sdk.NewInt64Coin(StateRentDenom, 0)
	}
	amount := billableBytes * p.RentRatePerBytePerBlock * blocks
	return sdk.NewInt64Coin(StateRentDenom, amount)
}
