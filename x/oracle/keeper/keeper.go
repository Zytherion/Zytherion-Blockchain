package keeper

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"zytherion/x/oracle/types"
)

// StakingKeeper defines the expected staking keeper interface used by the oracle keeper.
type StakingKeeper interface {
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (stakingtypes.Validator, bool)
	GetValidatorByConsAddr(ctx sdk.Context, consAddr sdk.ConsAddress) (stakingtypes.Validator, bool)
	IterateBondedValidatorsByPower(ctx sdk.Context, fn func(index int64, validator stakingtypes.ValidatorI) (stop bool))
}

// Keeper is the oracle module keeper.
type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	stakingKeeper StakingKeeper
}

// NewKeeper creates a new oracle Keeper.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	stakingKeeper StakingKeeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		stakingKeeper: stakingKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// StoreKey returns the module's store key (used by AppModule.EndBlock access).
func (k Keeper) StoreKey() storetypes.StoreKey {
	return k.storeKey
}

// GetStore returns the KVStore for the oracle module.
func (k Keeper) GetStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

// ─── Params ───────────────────────────────────────────────────────────────────

// GetParams returns the current oracle module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.OracleParams {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.OracleParamsKey))
	if bz == nil {
		return types.DefaultOracleParams()
	}
	var params types.OracleParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultOracleParams()
	}
	return params
}

// SetParams stores the oracle module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.OracleParams) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("oracle: failed to marshal params: %v", err))
	}
	store.Set([]byte(types.OracleParamsKey), bz)
}

// ─── Price storage ────────────────────────────────────────────────────────────

// priceKey builds the KVStore key for a given price entry.
// Format: price/<denom>/<height>
func priceKey(denom string, height int64) []byte {
	return []byte(fmt.Sprintf("%s%s/%d", types.PriceKeyPrefix, denom, height))
}

// SetPrice stores a PriceEntry in the KVStore.
func (k Keeper) SetPrice(ctx sdk.Context, entry types.PriceEntry) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("oracle: failed to marshal price entry: %w", err)
	}
	store.Set(priceKey(entry.Denom, entry.Height), bz)
	return nil
}

// GetLatestPrice iterates all stored prices for the given denom and returns
// the one with the highest block height.
func (k Keeper) GetLatestPrice(ctx sdk.Context, denom string) (types.PriceEntry, error) {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(fmt.Sprintf("%s%s/", types.PriceKeyPrefix, denom))

	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var latest types.PriceEntry
	found := false
	for ; iterator.Valid(); iterator.Next() {
		var entry types.PriceEntry
		if err := json.Unmarshal(iterator.Value(), &entry); err != nil {
			continue
		}
		if !found || entry.Height > latest.Height {
			latest = entry
			found = true
		}
	}

	if !found {
		return types.PriceEntry{}, fmt.Errorf("no price found for denom %s", denom)
	}
	return latest, nil
}

// GetPriceHistory returns all price entries for the given denom at or after fromHeight.
func (k Keeper) GetPriceHistory(ctx sdk.Context, denom string, fromHeight int64) []types.PriceEntry {
	store := ctx.KVStore(k.storeKey)
	prefix := []byte(fmt.Sprintf("%s%s/", types.PriceKeyPrefix, denom))

	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var entries []types.PriceEntry
	for ; iterator.Valid(); iterator.Next() {
		var entry types.PriceEntry
		if err := json.Unmarshal(iterator.Value(), &entry); err != nil {
			continue
		}
		if entry.Height >= fromHeight {
			entries = append(entries, entry)
		}
	}
	return entries
}

// ─── TWAP computation ─────────────────────────────────────────────────────────

// ComputeTWAP computes the median price (used as TWAP) from all price entries
// within the TwapWindowBlocks window. Returns ErrInsufficientSubmissions if
// the number of samples is below params.MinSubmissions.
func (k Keeper) ComputeTWAP(ctx sdk.Context, denom string) (types.TWAPData, error) {
	params := k.GetParams(ctx)
	currentHeight := ctx.BlockHeight()
	windowStart := currentHeight - params.TwapWindowBlocks
	if windowStart < 0 {
		windowStart = 0
	}

	entries := k.GetPriceHistory(ctx, denom, windowStart)
	if len(entries) < params.MinSubmissions {
		return types.TWAPData{}, types.ErrInsufficientSubmissions.Wrapf(
			"got %d submissions, need %d for denom %s",
			len(entries), params.MinSubmissions, denom,
		)
	}

	// Sort entries by PriceUSD to find the median.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PriceUSD.LT(entries[j].PriceUSD)
	})

	n := len(entries)
	var median sdk.Dec
	if n%2 == 1 {
		median = entries[n/2].PriceUSD
	} else {
		// Average the two middle values.
		mid1 := entries[n/2-1].PriceUSD
		mid2 := entries[n/2].PriceUSD
		median = mid1.Add(mid2).Quo(sdk.NewDec(2))
	}

	return types.TWAPData{
		Denom:       denom,
		TWAP:        median,
		WindowStart: windowStart,
		WindowEnd:   currentHeight,
		NumSamples:  n,
	}, nil
}

// ─── TWAP cache ───────────────────────────────────────────────────────────────

// twapKey builds the KVStore key for a TWAP entry.
// Format: twap/<denom>
func twapKey(denom string) []byte {
	return []byte(fmt.Sprintf("%s%s", types.TWAPKeyPrefix, denom))
}

// GetTWAP retrieves the latest cached TWAP for a denom.
func (k Keeper) GetTWAP(ctx sdk.Context, denom string) (types.TWAPData, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(twapKey(denom))
	if bz == nil {
		return types.TWAPData{}, fmt.Errorf("no TWAP found for denom %s", denom)
	}
	var data types.TWAPData
	if err := json.Unmarshal(bz, &data); err != nil {
		return types.TWAPData{}, fmt.Errorf("oracle: failed to unmarshal TWAP: %w", err)
	}
	return data, nil
}

// SetTWAP stores the computed TWAP data for a denom.
func (k Keeper) SetTWAP(ctx sdk.Context, data types.TWAPData) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(data)
	if err != nil {
		k.Logger(ctx).Error("oracle: failed to marshal TWAP", "denom", data.Denom, "err", err)
		return
	}
	store.Set(twapKey(data.Denom), bz)
}

// ─── Pruning ──────────────────────────────────────────────────────────────────

// PruneOldPrices deletes all price entries older than (currentHeight - TwapWindowBlocks - MaxPriceAge).
func (k Keeper) PruneOldPrices(ctx sdk.Context, currentHeight int64) {
	params := k.GetParams(ctx)
	cutoff := currentHeight - params.TwapWindowBlocks - params.MaxPriceAge
	if cutoff <= 0 {
		return
	}

	store := ctx.KVStore(k.storeKey)
	prefix := []byte(types.PriceKeyPrefix)
	iterator := sdk.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var toDelete [][]byte
	for ; iterator.Valid(); iterator.Next() {
		var entry types.PriceEntry
		if err := json.Unmarshal(iterator.Value(), &entry); err != nil {
			continue
		}
		if entry.Height < cutoff {
			// Capture the key (iterator.Key() may be invalidated after Close)
			keyCopy := make([]byte, len(iterator.Key()))
			copy(keyCopy, iterator.Key())
			toDelete = append(toDelete, keyCopy)
		}
	}

	for _, key := range toDelete {
		store.Delete(key)
	}
}

// ─── Validator check ──────────────────────────────────────────────────────────

// IsValidatorSubmitter checks whether the given AccAddress corresponds to a
// currently bonded validator by iterating bonded validators by power.
func (k Keeper) IsValidatorSubmitter(ctx sdk.Context, addr sdk.AccAddress) bool {
	// Derive the expected operator (val) address from the account address.
	// We also accept the address as a validator operator address directly.
	valAddr := sdk.ValAddress(addr)
	if _, found := k.stakingKeeper.GetValidator(ctx, valAddr); found {
		return true
	}

	// Fallback: iterate all bonded validators and match by operator bech32 address.
	addrStr := addr.String()
	found := false
	k.stakingKeeper.IterateBondedValidatorsByPower(ctx, func(_ int64, val stakingtypes.ValidatorI) bool {
		// val.GetOperator() returns a ValAddress; convert to AccAddress to compare.
		valOpAddr := sdk.AccAddress(val.GetOperator())
		if strings.EqualFold(valOpAddr.String(), addrStr) {
			found = true
			return true // stop
		}
		return false
	})
	return found
}
