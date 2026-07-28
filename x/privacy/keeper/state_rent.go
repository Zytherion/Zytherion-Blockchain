package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"zytherion/x/privacy/types"
)

const (
	// stateRentParamsKey is the KV store key for StateRentParams.
	stateRentParamsKey = "state_rent_params"

	// rentLastChargedPrefix stores the last block at which rent was charged
	// for each commitment key, format: "rent_last/<commitmentKey>"
	rentLastChargedPrefix = "rent_last/"

	// rentGraceStartPrefix stores the block height at which a commitment
	// entered the grace period, format: "rent_grace/<commitmentKey>"
	rentGraceStartPrefix = "rent_grace/"
)

// GetStateRentParams returns current state rent parameters.
// Falls back to defaults if never set (e.g., before first governance proposal).
func (k Keeper) GetStateRentParams(ctx sdk.Context) types.StateRentParams {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(stateRentParamsKey))
	if bz == nil {
		return types.DefaultStateRentParams()
	}
	var params types.StateRentParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultStateRentParams()
	}
	return params
}

// SetStateRentParams stores updated state rent parameters.
// Called by governance parameter change proposals.
func (k Keeper) SetStateRentParams(ctx sdk.Context, params types.StateRentParams) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("privacy: marshal state rent params: %w", err)
	}
	store.Set([]byte(stateRentParamsKey), bz)
	return nil
}

// CollectRent charges rent for a single encrypted commitment.
//
// commitmentKey: the KV store key of the encrypted blob (string form).
// owner: the address responsible for paying rent.
// sizeBytes: current size of the encrypted data in bytes.
//
// Returns:
//   - nil if rent was collected successfully (or storage is in free tier).
//   - an error describing the failure if the owner cannot pay.
//
// On payment failure, the commitment enters (or remains in) its grace period.
// Use CheckAndEvict to prune commitments whose grace period has expired.
func (k Keeper) CollectRent(
	ctx sdk.Context,
	commitmentKey string,
	owner sdk.AccAddress,
	sizeBytes int64,
) error {
	params := k.GetStateRentParams(ctx)
	currentHeight := ctx.BlockHeight()

	// Determine blocks since last charge.
	lastCharged := k.getRentLastCharged(ctx, commitmentKey)
	blocksSinceCharge := currentHeight - lastCharged
	if blocksSinceCharge <= 0 {
		return nil // already charged this block
	}

	// Calculate rent due.
	rentDue := params.RentDue(sizeBytes, blocksSinceCharge)
	if rentDue.IsZero() {
		// Free tier — update the last-charged height without billing.
		k.setRentLastCharged(ctx, commitmentKey, currentHeight)
		return nil
	}

	// Attempt to collect rent from the owner's account.
	err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		owner,
		types.ModuleName,
		sdk.NewCoins(rentDue),
	)
	if err != nil {
		// Owner cannot pay — enter or extend grace period.
		k.handleRentDefault(ctx, commitmentKey, owner, currentHeight, params)
		return fmt.Errorf("privacy: rent collection failed for %s: %w", commitmentKey, err)
	}

	// Success — burn the collected rent (deflationary).
	// Governance can modify this policy to redirect to a community pool instead.
	_ = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(rentDue))

	// Update last-charged height.
	k.setRentLastCharged(ctx, commitmentKey, currentHeight)

	// Clear any outstanding grace period now that the owner has caught up.
	k.clearGracePeriod(ctx, commitmentKey)

	// Emit event for on-chain transparency.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRentCollected,
		sdk.NewAttribute("commitment_key", commitmentKey),
		sdk.NewAttribute("owner", owner.String()),
		sdk.NewAttribute("amount", rentDue.String()),
		sdk.NewAttribute("size_bytes", fmt.Sprintf("%d", sizeBytes)),
	))

	return nil
}

// CheckAndEvict checks if a commitment has exceeded its grace period
// and should be pruned. Returns true if the commitment was evicted.
//
// Call this from EndBlocker for each commitment currently in a grace period.
func (k Keeper) CheckAndEvict(
	ctx sdk.Context,
	commitmentKey string,
) bool {
	graceStart := k.getGraceStart(ctx, commitmentKey)
	if graceStart == 0 {
		return false // not in grace period
	}

	params := k.GetStateRentParams(ctx)
	if ctx.BlockHeight()-graceStart < params.GracePeriodBlocks {
		return false // still within grace period — give owner more time
	}

	// Grace period expired — emit archival event BEFORE eviction
	// so external archival nodes (Arweave/Filecoin) have a chance to
	// capture the data. They should have been watching since graceStart.
	store := ctx.KVStore(k.storeKey)
	blobData := store.Get([]byte(commitmentKey))
	if blobData != nil {
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeCommitmentEvicted,
			sdk.NewAttribute("commitment_key", commitmentKey),
			sdk.NewAttribute("grace_started_at", fmt.Sprintf("%d", graceStart)),
			sdk.NewAttribute("evicted_at", fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute("data_size_bytes", fmt.Sprintf("%d", len(blobData))),
			// NOTE: actual blob data NOT included in event — too large.
			// Archival nodes should have been tracking since grace_started_at.
		))

		// Prune the commitment data.
		store.Delete([]byte(commitmentKey))
	}

	// Clean up rent tracking keys.
	store.Delete([]byte(rentLastChargedPrefix + commitmentKey))
	store.Delete([]byte(rentGraceStartPrefix + commitmentKey))

	return true
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (k Keeper) getRentLastCharged(ctx sdk.Context, commitmentKey string) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(rentLastChargedPrefix + commitmentKey))
	if bz == nil {
		// First time we see this commitment — treat the current block as the
		// start of billing so we don't back-charge for historical blocks.
		return ctx.BlockHeight()
	}
	var h int64
	if err := json.Unmarshal(bz, &h); err != nil {
		return ctx.BlockHeight()
	}
	return h
}

func (k Keeper) setRentLastCharged(ctx sdk.Context, commitmentKey string, height int64) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(height)
	store.Set([]byte(rentLastChargedPrefix+commitmentKey), bz)
}

func (k Keeper) handleRentDefault(
	ctx sdk.Context,
	commitmentKey string,
	owner sdk.AccAddress,
	currentHeight int64,
	params types.StateRentParams,
) {
	// Only start grace period if not already started.
	if k.getGraceStart(ctx, commitmentKey) == 0 {
		k.setGraceStart(ctx, commitmentKey, currentHeight)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			types.EventTypeRentDefault,
			sdk.NewAttribute("commitment_key", commitmentKey),
			sdk.NewAttribute("owner", owner.String()),
			sdk.NewAttribute("grace_period_blocks", fmt.Sprintf("%d", params.GracePeriodBlocks)),
			sdk.NewAttribute("eviction_at_block", fmt.Sprintf("%d", currentHeight+params.GracePeriodBlocks)),
		))
	}
}

func (k Keeper) getGraceStart(ctx sdk.Context, commitmentKey string) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(rentGraceStartPrefix + commitmentKey))
	if bz == nil {
		return 0
	}
	var h int64
	_ = json.Unmarshal(bz, &h)
	return h
}

func (k Keeper) setGraceStart(ctx sdk.Context, commitmentKey string, height int64) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(height)
	store.Set([]byte(rentGraceStartPrefix+commitmentKey), bz)
}

func (k Keeper) clearGracePeriod(ctx sdk.Context, commitmentKey string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete([]byte(rentGraceStartPrefix + commitmentKey))
}
