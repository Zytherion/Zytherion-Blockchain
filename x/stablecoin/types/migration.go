// Package types — Migration Grace Period State Machine for ZYTD.
//
// # 3-Phase Lock-Only Grace Period
//
// When the Dilithium5 PQC key requirement is activated, existing users need time
// to register their new post-quantum keys without losing access to their positions.
// This state machine governs what operations are permitted at each chain height.
//
// ┌─────────────────────────────────────────────────────────────────────────────┐
// │  Phase 1: Active Registration   height <= GracePhaseLockHeight (20 000)    │
// │    • All operations allowed: Mint, Burn, Liquidate, RegisterKey              │
// │    • Users are encouraged (but not forced) to register Dilithium5 keys       │
// ├─────────────────────────────────────────────────────────────────────────────┤
// │  Phase 2: Lock-Only Migration   height in (20 000, 50 000]                  │
// │    • Minting NEW ZYTD is FORBIDDEN → ErrMigrationMintForbidden              │
// │    • Burn and RegisterKey are still allowed (exit-only)                      │
// │    • Liquidations still allowed (protocol health)                            │
// ├─────────────────────────────────────────────────────────────────────────────┤
// │  Phase 3: Hard Enforcement      height > GracePhaseHardHeight (50 000)      │
// │    • All ZYTD operations REQUIRE a registered Dilithium5 key                 │
// │    • Any operation without a valid registered key → ErrHardFreezeNoKey      │
// │    • Any operation with an invalid/missing Dilithium5 sig → ErrHardFreeze   │
// └─────────────────────────────────────────────────────────────────────────────┘
package types

import errorsmod "cosmossdk.io/errors"

// ── Phase boundary heights ───────────────────────────────────────────────────

const (
	// GracePhaseLockHeight is the block height at which Phase 1 ends and
	// Phase 2 (lock-only) begins. Mint operations are forbidden above this.
	GracePhaseLockHeight int64 = 20_000

	// GracePhaseHardHeight is the block height at which Phase 2 ends and
	// Phase 3 (hard enforcement) begins. All operations require Dilithium5.
	GracePhaseHardHeight int64 = 50_000
)

// ── Migration phase errors ───────────────────────────────────────────────────

var (
	// ErrMigrationMintForbidden is returned in Phase 2 when a user attempts
	// to mint new ZYTD before registering a Dilithium5 key.
	// Burn and RegisterKey are still allowed during this phase.
	ErrMigrationMintForbidden = errorsmod.Register(
		ModuleName, 1610,
		"ZYTD minting is forbidden during key migration window (Phase 2). "+
			"Please register your Dilithium5 key via MsgRegisterZYTDKey, then burn existing positions to migrate",
	)

	// ErrHardFreezeNoKey is returned in Phase 3 when no Dilithium5 public key
	// has been registered for the sender address.
	ErrHardFreezeNoKey = errorsmod.Register(
		ModuleName, 1611,
		"HARD FREEZE: no Dilithium5 key registered for this address. "+
			"All ZYTD operations require a registered post-quantum key (Phase 3 enforcement active)",
	)

	// ErrHardFreeze is returned in Phase 3 when a Dilithium5 key IS registered
	// but the message carries no valid Dilithium5 signature.
	ErrHardFreeze = errorsmod.Register(
		ModuleName, 1612,
		"HARD FREEZE: invalid or missing Dilithium5 signature. "+
			"All ZYTD operations must include a valid Dilithium5 signature (Phase 3 enforcement active)",
	)
)

// ── MigrationPhase type ──────────────────────────────────────────────────────

// MigrationPhase represents the current migration enforcement phase.
type MigrationPhase uint8

const (
	// MigrationPhase1 — active registration: all operations allowed.
	MigrationPhase1 MigrationPhase = 1

	// MigrationPhase2 — lock-only: mint forbidden, burn/register allowed.
	MigrationPhase2 MigrationPhase = 2

	// MigrationPhase3 — hard enforcement: Dilithium5 required for all ops.
	MigrationPhase3 MigrationPhase = 3
)

// GetMigrationPhase returns the current MigrationPhase for the given block height.
//
//	height <= 20 000           → Phase 1 (open)
//	20 000 < height <= 50 000  → Phase 2 (lock-only migration)
//	height > 50 000            → Phase 3 (hard enforcement)
func GetMigrationPhase(currentHeight int64) MigrationPhase {
	switch {
	case currentHeight <= GracePhaseLockHeight:
		return MigrationPhase1
	case currentHeight <= GracePhaseHardHeight:
		return MigrationPhase2
	default:
		return MigrationPhase3
	}
}

// CheckMintAllowed returns an error if minting is forbidden at the given height.
// Call this at the START of MsgMintZYTD processing.
func CheckMintAllowed(currentHeight int64) error {
	if GetMigrationPhase(currentHeight) == MigrationPhase2 {
		return ErrMigrationMintForbidden
	}
	return nil
}

// CheckHardFreeze returns an error if Phase 3 is active and the given
// hasPQCKey flag is false (no Dilithium5 key registered).
// Call this for ALL ZYTD operations in Phase 3.
func CheckHardFreeze(currentHeight int64, hasPQCKey bool) error {
	if GetMigrationPhase(currentHeight) != MigrationPhase3 {
		return nil // Phase 1 or 2: no hard freeze
	}
	if !hasPQCKey {
		return ErrHardFreezeNoKey
	}
	return nil
}

// String returns a human-readable phase name for logging and events.
func (p MigrationPhase) String() string {
	switch p {
	case MigrationPhase1:
		return "Phase1-ActiveRegistration"
	case MigrationPhase2:
		return "Phase2-LockOnlyMigration"
	case MigrationPhase3:
		return "Phase3-HardEnforcement"
	default:
		return "Unknown"
	}
}
