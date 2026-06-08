// query_tfhe_result.go — TFHE on-demand reconstruction and result query handler.
//
// Endpoint: query/tfhe_result
//
// Process:
//  1. Feature flag check.
//  2. Look up shard metadata for the requested commitment.
//  3. Attempt to load the result from the result store (cached from previous evaluation).
//  4. If not cached: reconstruct the ciphertext from shards (on-demand).
//  5. Return the ciphertext bytes (base64-encoded in the response).
package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"zytherion/x/privacy/types"
)

// TFHEResult implements the query/tfhe_result gRPC endpoint.
func (k Keeper) TFHEResult(
	goCtx context.Context,
	req *types.QueryTFHEResultRequest,
) (*types.QueryTFHEResultResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request: nil")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// ── 1. Feature flag check ──────────────────────────────────────────────────
	if !k.IsTFHEEnabled() {
		return nil, status.Error(codes.FailedPrecondition,
			types.ErrTFHEDisabled.Error())
	}

	// ── 2. Validate commitment hash ────────────────────────────────────────────
	if len(req.CommitmentHash) != 32 {
		return nil, status.Errorf(codes.InvalidArgument,
			"commitment hash must be 32 bytes, got %d", len(req.CommitmentHash))
	}

	// ── 3. Check result cache ──────────────────────────────────────────────────
	if resultCt, ok := k.GetTFHEResult(ctx, req.CommitmentHash); ok {
		k.Logger(ctx).Debug("tfhe_result: cache hit",
			"commitment", fmt.Sprintf("%x", req.CommitmentHash[:4]))
		return &types.QueryTFHEResultResponse{
			CommitmentHash:    req.CommitmentHash,
			ResultCiphertext:  resultCt,
			ReconstructedFrom: 0, // from cache — no shard reconstruction needed
		}, nil
	}

	// ── 4. Look up shard metadata ──────────────────────────────────────────────
	meta, ok := k.GetTFHEMeta(ctx, req.CommitmentHash)
	if !ok {
		return nil, status.Errorf(codes.NotFound,
			"no TFHE ciphertext found for commitment %x", req.CommitmentHash)
	}

	// ── 5. Reconstruct ciphertext from shards ──────────────────────────────────
	reconstructed, err := k.shardDistributor.ReconstructCiphertext(ctx.Context(), meta)
	if err != nil {
		return nil, status.Errorf(codes.Internal,
			"shard reconstruction failed: %v", err)
	}

	k.Logger(ctx).Info("tfhe_result: reconstructed ciphertext from shards",
		"commitment", fmt.Sprintf("%x", req.CommitmentHash[:4]),
		"ciphertext_bytes", len(reconstructed),
	)

	// ── 6. Cache the result ────────────────────────────────────────────────────
	k.SetTFHEResult(ctx, req.CommitmentHash, reconstructed)

	return &types.QueryTFHEResultResponse{
		CommitmentHash:    req.CommitmentHash,
		ResultCiphertext:  reconstructed,
		ReconstructedFrom: uint32(meta.OriginalLen),
	}, nil
}
