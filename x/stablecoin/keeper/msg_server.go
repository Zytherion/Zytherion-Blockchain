package keeper

import (
	"bytes"
	"context"
	"golang.org/x/crypto/sha3"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/pqc"
	"zytherion/x/stablecoin/types"
)

type mintMsgServer struct {
	Keeper
}

// NewMsgServerImpl returns a new message server implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &mintMsgServer{Keeper: keeper}
}

// ── PQC helpers ──────────────────────────────────────────────────────────────

// buildZYTDMsgHash constructs the deterministic message hash that the sender
// must sign with their Dilithium5 private key.
//
// Format: SHA3-256(domain || sender_bytes || ibc_denom || expiration_height_LE8)
// The expiration_block_height is included to ensure forward-only replay protection
// (each signature is only valid for a specific target block range).
func buildZYTDMsgHash(sender sdk.AccAddress, ibcDenom string, expirationBlockHeight int64) []byte {
	h := sha3.New256()
	h.Write([]byte("zytherion/zytd/v1/")) // domain separator
	h.Write(sender.Bytes())
	h.Write([]byte(ibcDenom))
	heightBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBuf, uint64(expirationBlockHeight))
	h.Write(heightBuf)
	return h.Sum(nil)
}

// verifyZYTDPQCSig verifies a Dilithium5 signature on a ZYTD message.
//
// It performs three checks in order:
//  1. Phase 3 enforcement — if height > 50000 and no key is registered → ErrHardFreezeNoKey
//  2. If a PQC key IS registered, verify the provided pubkey matches it → ErrPQCKeyMismatch
//  3. Verify the Dilithium5 sig over the canonical message hash → ErrInvalidPQCSig
//
// In Phase 1 and Phase 2 this function is a no-op if no key is registered,
// giving users time to adopt PQC keys without breaking existing flows.
func (s mintMsgServer) verifyZYTDPQCSig(
	ctx sdk.Context,
	sender sdk.AccAddress,
	ibcDenom string,
	expiration int64,
	providedPubKey []byte,
	providedSig []byte,
) error {
	currentHeight := ctx.BlockHeight()
	phase := types.GetMigrationPhase(currentHeight)

	registeredKey, err := s.GetZYTDPQCKey(ctx, sender)
	hasKey := err == nil

	// Phase 3 hard enforcement: key MUST be registered.
	if err := types.CheckHardFreeze(currentHeight, hasKey); err != nil {
		return err
	}

	// If no key is registered and we're in Phase 1 or 2, skip PQC checks.
	if !hasKey {
		if phase == types.MigrationPhase1 || phase == types.MigrationPhase2 {
			return nil // PQC not yet required
		}
	}

	// Key is registered: verify the provided pubkey matches what's on-chain.
	if !bytes.Equal(registeredKey, providedPubKey) {
		return types.ErrPQCKeyMismatch
	}

	// Verify Dilithium5 signature over the canonical message hash.
	msgHash := buildZYTDMsgHash(sender, ibcDenom, expiration)
	if !pqc.Verify(msgHash, providedSig, providedPubKey) {
		if phase == types.MigrationPhase3 {
			return types.ErrHardFreeze
		}
		return types.ErrInvalidPQCSig
	}
	return nil
}

// MintZYTD handles MsgMintZYTD: lock collateral and mint ZYTD stablecoin.
// Steps:
//  1. Get CollateralAsset from ibcCollateralKeeper — error if not whitelisted/active.
//  2. Get TWAP price from oracleKeeper — error if unavailable.
//  3. Calculate collateralUSD = collateralAmount * TWAPprice.
//  4. Calculate maxMintable = collateralUSD / asset.MinRatio.
//  5. Error ErrExceedsCollateralValue if requestedZYTD > maxMintable.
//  6. Lock collateral via ibcCollateralKeeper.
//  7. Mint ZYTD via bankKeeper.
//  8. Send minted coins to sender.
//  9. Update MintedZYTD and store MintRecord.
// 10. Emit event.
func (s mintMsgServer) MintZYTD(goCtx context.Context, req *types.MsgMintZYTD) (*types.MsgMintZYTDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(req.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// ═ Phase check: minting is FORBIDDEN in Phase 2 (lock-only migration). ═
	if err := types.CheckMintAllowed(ctx.BlockHeight()); err != nil {
		return nil, err
	}

	// ═ PQC signature verification + Phase 3 hard freeze check. ═
	if err := s.verifyZYTDPQCSig(ctx, sender, req.IBCDenom, req.ExpirationBlockHeight, req.Dilithium5PubKey, req.Dilithium5Sig); err != nil {
		return nil, err
	}

	// Step 1: Get and validate collateral asset
	asset, err := s.ibcCollateralKeeper.GetCollateralAsset(ctx, req.IBCDenom)
	if err != nil {
		return nil, fmt.Errorf("collateral asset not found: %w", err)
	}
	if !asset.IsActive {
		return nil, fmt.Errorf("collateral asset %s is not active", req.IBCDenom)
	}

	// Step 2: Get TWAP price
	twap, err := s.oracleKeeper.GetTWAP(ctx, req.IBCDenom)
	if err != nil {
		return nil, types.ErrOraclePriceUnavailable
	}

	// Step 3: Calculate collateral USD value
	collateralUSD := sdk.NewDecFromInt(req.CollateralAmount).Mul(twap.TWAP)

	// Step 4: Calculate max mintable ZYTD
	maxMintable := collateralUSD.Quo(asset.MinRatio).TruncateInt()

	// Step 5: Check requested amount is within limits
	if req.RequestedZYTD.GT(maxMintable) {
		return nil, types.ErrExceedsCollateralValue
	}

	// Step 6: Lock collateral
	if err := s.ibcCollateralKeeper.LockCollateral(ctx, sender, req.IBCDenom, req.CollateralAmount); err != nil {
		return nil, fmt.Errorf("lock collateral: %w", err)
	}

	// Step 7: Mint ZYTD coins to module account
	mintCoins := sdk.NewCoins(sdk.NewCoin(types.ZYTDDenom, req.RequestedZYTD))
	if err := s.bankKeeper.MintCoins(ctx, types.ModuleAccountName, mintCoins); err != nil {
		return nil, fmt.Errorf("mint ZYTD: %w", err)
	}

	// Step 8: Send minted coins from module to sender
	if err := s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleAccountName, sender, mintCoins); err != nil {
		return nil, fmt.Errorf("send minted ZYTD to sender: %w", err)
	}

	// Step 9a: Update MintedZYTD in position
	if err := s.ibcCollateralKeeper.UpdateMintedZYTD(ctx, sender, req.IBCDenom, req.RequestedZYTD); err != nil {
		return nil, fmt.Errorf("update minted ZYTD: %w", err)
	}

	// Step 9b: Store MintRecord
	record := types.MintRecord{
		Owner:         req.Sender,
		IBCDenom:      req.IBCDenom,
		Minted:        req.RequestedZYTD,
		CollateralUSD: collateralUSD,
		MintedAt:      ctx.BlockHeight(),
	}
	// Check if there's an existing record to accumulate
	existing, err := s.GetMintRecord(ctx, sender, req.IBCDenom)
	if err == nil {
		record.Minted = existing.Minted.Add(req.RequestedZYTD)
		record.CollateralUSD = existing.CollateralUSD.Add(collateralUSD)
		record.MintedAt = existing.MintedAt // preserve original mint height
	}
	s.SetMintRecord(ctx, record)

	// Update total supply
	supply := s.GetTotalSupply(ctx)
	s.SetTotalSupply(ctx, supply.Add(req.RequestedZYTD))

	// Step 10: Emit event
	collateralRatio := collateralUSD.Quo(sdk.NewDecFromInt(req.RequestedZYTD))
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeMintZYTD,
		sdk.NewAttribute(types.AttributeKeyOwner, req.Sender),
		sdk.NewAttribute(types.AttributeKeyIBCDenom, req.IBCDenom),
		sdk.NewAttribute(types.AttributeKeyCollateralAmount, req.CollateralAmount.String()),
		sdk.NewAttribute(types.AttributeKeyZYTDAmount, req.RequestedZYTD.String()),
		sdk.NewAttribute(types.AttributeKeyCollateralRatio, collateralRatio.String()),
	))

	return &types.MsgMintZYTDResponse{MintedAmount: req.RequestedZYTD}, nil
}

// BurnZYTD handles MsgBurnZYTD: burn ZYTD and unlock proportional collateral.
// Steps:
//  1. Get MintRecord for sender+ibcDenom — error if not found.
//  2. Error if zytdAmount > MintRecord.Minted.
//  3. Send ZYTD from sender to module account and burn.
//  4. Calculate proportional collateral to return.
//  5. Unlock collateral via ibcCollateralKeeper.
//  6. Update ibcCollateralKeeper.UpdateMintedZYTD with negative delta.
//  7. Update or delete MintRecord.
//  8. Update TotalSupply.
//  9. Emit event.
func (s mintMsgServer) BurnZYTD(goCtx context.Context, req *types.MsgBurnZYTD) (*types.MsgBurnZYTDResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(req.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// ═ PQC signature verification + Phase 3 hard freeze check. ═
	// Burn is allowed in all phases but Phase 3 requires a valid sig.
	if err := s.verifyZYTDPQCSig(ctx, sender, req.IBCDenom, req.ExpirationBlockHeight, req.Dilithium5PubKey, req.Dilithium5Sig); err != nil {
		return nil, err
	}

	// Step 1: Get MintRecord
	record, err := s.GetMintRecord(ctx, sender, req.IBCDenom)
	if err != nil {
		return nil, types.ErrMintRecordNotFound
	}

	// Step 2: Validate burn amount
	if req.ZYTDAmount.GT(record.Minted) {
		return nil, types.ErrExceedsMintedAmount
	}

	// Step 3: Send ZYTD from sender to module and burn
	burnCoins := sdk.NewCoins(sdk.NewCoin(types.ZYTDDenom, req.ZYTDAmount))
	if err := s.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleAccountName, burnCoins); err != nil {
		return nil, fmt.Errorf("send ZYTD to module: %w", err)
	}
	if err := s.bankKeeper.BurnCoins(ctx, types.ModuleAccountName, burnCoins); err != nil {
		return nil, fmt.Errorf("burn ZYTD: %w", err)
	}

	// Step 4: Calculate proportional collateral to return
	// returnAmount = (zytdAmount / MintRecord.Minted) * lockedCollateral
	position, err := s.ibcCollateralKeeper.GetPosition(ctx, sender, req.IBCDenom)
	if err != nil {
		return nil, fmt.Errorf("get collateral position: %w", err)
	}
	burnRatio := sdk.NewDecFromInt(req.ZYTDAmount).Quo(sdk.NewDecFromInt(record.Minted))
	returnAmount := burnRatio.MulInt(position.Amount).TruncateInt()

	// Step 5: Unlock proportional collateral
	if err := s.ibcCollateralKeeper.UnlockCollateral(ctx, sender, req.IBCDenom, returnAmount); err != nil {
		return nil, fmt.Errorf("unlock collateral: %w", err)
	}

	// Step 6: Update MintedZYTD in position (negative delta)
	negDelta := req.ZYTDAmount.Neg()
	if err := s.ibcCollateralKeeper.UpdateMintedZYTD(ctx, sender, req.IBCDenom, negDelta); err != nil {
		return nil, fmt.Errorf("update minted ZYTD: %w", err)
	}

	// Step 7: Update or delete MintRecord
	newMinted := record.Minted.Sub(req.ZYTDAmount)
	if newMinted.IsZero() {
		s.DeleteMintRecord(ctx, sender, req.IBCDenom)
	} else {
		record.Minted = newMinted
		record.CollateralUSD = record.CollateralUSD.Mul(sdk.OneDec().Sub(burnRatio))
		s.SetMintRecord(ctx, record)
	}

	// Step 8: Update TotalSupply
	supply := s.GetTotalSupply(ctx)
	newSupply := supply.Sub(req.ZYTDAmount)
	if newSupply.IsNegative() {
		newSupply = sdk.ZeroInt()
	}
	s.SetTotalSupply(ctx, newSupply)

	// Step 9: Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeBurnZYTD,
		sdk.NewAttribute(types.AttributeKeyOwner, req.Sender),
		sdk.NewAttribute(types.AttributeKeyIBCDenom, req.IBCDenom),
		sdk.NewAttribute(types.AttributeKeyZYTDAmount, req.ZYTDAmount.String()),
		sdk.NewAttribute(types.AttributeKeyCollateralAmount, returnAmount.String()),
	))

	return &types.MsgBurnZYTDResponse{UnlockedAmount: returnAmount}, nil
}

// Liquidate handles MsgLiquidate: liquidate an undercollateralized position.
// Steps:
//  1. Get CollateralPosition of targetOwner+ibcDenom.
//  2. Get TWAP price from oracleKeeper.
//  3. Calculate currentRatio = (collateralAmount * price) / MintedZYTD.
//  4. Error ErrPositionHealthy if currentRatio >= asset.LiquidationThreshold.
//  5. Calculate collateral to seize, liquidator reward, and protocol fee.
//  6. Liquidator sends debtZYTD to module → burned.
//  7. Transfer collateral rewards.
//  8. Close position.
//  9. Emit event.
func (s mintMsgServer) Liquidate(goCtx context.Context, req *types.MsgLiquidate) (*types.MsgLiquidateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	liquidator, err := sdk.AccAddressFromBech32(req.Liquidator)
	if err != nil {
		return nil, fmt.Errorf("invalid liquidator address: %w", err)
	}
	targetOwner, err := sdk.AccAddressFromBech32(req.TargetOwner)
	if err != nil {
		return nil, fmt.Errorf("invalid target_owner address: %w", err)
	}

	// ═ PQC signature verification + Phase 3 hard freeze check for liquidator. ═
	// Liquidations allowed in all phases but Phase 3 requires valid sig from liquidator.
	if err := s.verifyZYTDPQCSig(ctx, liquidator, req.IBCDenom, req.ExpirationBlockHeight, req.Dilithium5PubKey, req.Dilithium5Sig); err != nil {
		return nil, err
	}

	// Step 1: Get target's collateral position
	position, err := s.ibcCollateralKeeper.GetPosition(ctx, targetOwner, req.IBCDenom)
	if err != nil {
		return nil, fmt.Errorf("get target position: %w", err)
	}

	// Get collateral asset for thresholds
	asset, err := s.ibcCollateralKeeper.GetCollateralAsset(ctx, req.IBCDenom)
	if err != nil {
		return nil, fmt.Errorf("get collateral asset: %w", err)
	}

	// Step 2: Get TWAP price
	twap, err := s.oracleKeeper.GetTWAP(ctx, req.IBCDenom)
	if err != nil {
		return nil, types.ErrOraclePriceUnavailable
	}

	// Step 3: Calculate current collateral ratio
	collateralValue := sdk.NewDecFromInt(position.Amount).Mul(twap.TWAP)
	if position.MintedZYTD.IsZero() {
		return nil, types.ErrPositionHealthy
	}
	currentRatio := collateralValue.Quo(sdk.NewDecFromInt(position.MintedZYTD))

	// Step 4: Check if position is underwater
	if currentRatio.GTE(asset.LiquidationThreshold) {
		return nil, types.ErrPositionHealthy
	}

	params := s.GetParams(ctx)

	// Step 5: Calculate amounts
	debtZYTD := position.MintedZYTD
	// collateralToSeize = (debtZYTD / price) * (1 + LiquidationPenalty)
	debtInCollateral := sdk.NewDecFromInt(debtZYTD).Quo(twap.TWAP)
	collateralToSeize := debtInCollateral.Mul(sdk.OneDec().Add(params.LiquidationPenalty)).TruncateInt()
	// Cap at actual position amount
	if collateralToSeize.GT(position.Amount) {
		collateralToSeize = position.Amount
	}

	// liquidatorReward = debtInCollateral * LiquidatorReward
	liquidatorReward := debtInCollateral.Mul(params.LiquidatorReward).TruncateInt()
	if liquidatorReward.GT(collateralToSeize) {
		liquidatorReward = collateralToSeize
	}
	protocolFee := collateralToSeize.Sub(liquidatorReward)

	// Step 6: Liquidator burns debtZYTD
	debtCoins := sdk.NewCoins(sdk.NewCoin(types.ZYTDDenom, debtZYTD))
	if err := s.bankKeeper.SendCoinsFromAccountToModule(ctx, liquidator, types.ModuleAccountName, debtCoins); err != nil {
		return nil, fmt.Errorf("liquidator send ZYTD: %w", err)
	}
	if err := s.bankKeeper.BurnCoins(ctx, types.ModuleAccountName, debtCoins); err != nil {
		return nil, fmt.Errorf("burn liquidated ZYTD: %w", err)
	}

	// Step 7: Unlock collateral to seize and distribute
	if err := s.ibcCollateralKeeper.UnlockCollateral(ctx, targetOwner, req.IBCDenom, collateralToSeize); err != nil {
		return nil, fmt.Errorf("unlock seized collateral: %w", err)
	}

	// Transfer liquidator reward from module to liquidator
	if liquidatorReward.IsPositive() {
		rewardCoins := sdk.NewCoins(sdk.NewCoin(req.IBCDenom, liquidatorReward))
		_ = s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleAccountName, liquidator, rewardCoins)
	}

	// Transfer protocol fee to ProtocolFeeReceiver
	if protocolFee.IsPositive() {
		feeReceiver, err := sdk.AccAddressFromBech32(params.ProtocolFeeReceiver)
		if err == nil {
			feeCoins := sdk.NewCoins(sdk.NewCoin(req.IBCDenom, protocolFee))
			_ = s.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleAccountName, feeReceiver, feeCoins)
		}
	}

	// Step 8: Close position — zero out MintedZYTD
	negDebt := debtZYTD.Neg()
	if err := s.ibcCollateralKeeper.UpdateMintedZYTD(ctx, targetOwner, req.IBCDenom, negDebt); err != nil {
		return nil, fmt.Errorf("close liquidated position: %w", err)
	}
	s.DeleteMintRecord(ctx, targetOwner, req.IBCDenom)

	// Update total supply
	supply := s.GetTotalSupply(ctx)
	newSupply := supply.Sub(debtZYTD)
	if newSupply.IsNegative() {
		newSupply = sdk.ZeroInt()
	}
	s.SetTotalSupply(ctx, newSupply)

	// Step 9: Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeLiquidation,
		sdk.NewAttribute(types.AttributeKeyLiquidator, req.Liquidator),
		sdk.NewAttribute(types.AttributeKeyTarget, req.TargetOwner),
		sdk.NewAttribute(types.AttributeKeyIBCDenom, req.IBCDenom),
		sdk.NewAttribute(types.AttributeKeyZYTDAmount, debtZYTD.String()),
		sdk.NewAttribute(types.AttributeKeyCollateralAmount, collateralToSeize.String()),
	))

	return &types.MsgLiquidateResponse{SeizedAmount: collateralToSeize}, nil
}

// ═ RegisterZYTDKey ════════════════════════════════════════════════════════════

// RegisterZYTDKey handles MsgRegisterZYTDKey.
// Registers (or rotates) a Dilithium5 public key for the sender address.
// This operation is allowed in ALL migration phases (Phase 1, 2, and 3).
func (s mintMsgServer) RegisterZYTDKey(
	goCtx context.Context,
	req *types.MsgRegisterZYTDKey,
) (*types.MsgRegisterZYTDKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(req.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	// Store the Dilithium5 public key in the registry.
	// ValidateBasic already confirmed len == 2592.
	s.SetZYTDPQCKey(ctx, sender, req.Dilithium5PubKey)

	// Compute a SHA-256 fingerprint of the registered key for the response.
	h := sha3.New256()
	h.Write(req.Dilithium5PubKey)
	pubKeyHash := hex.EncodeToString(h.Sum(nil))

	phase := types.GetMigrationPhase(ctx.BlockHeight())

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"register_zytd_key",
		sdk.NewAttribute("sender", req.Sender),
		sdk.NewAttribute("dilithium5_key_hash", pubKeyHash),
		sdk.NewAttribute("migration_phase", phase.String()),
		sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
	))

	s.Logger(ctx).Info("ZYTD Dilithium5 key registered",
		"sender", req.Sender,
		"key_hash", pubKeyHash,
		"phase", phase.String(),
	)

	return &types.MsgRegisterZYTDKeyResponse{RegisteredPubKeyHash: pubKeyHash}, nil
}

// Compile-time assertion: mintMsgServer must fully implement types.MsgServer.
var _ types.MsgServer = (*mintMsgServer)(nil)
