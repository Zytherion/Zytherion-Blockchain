// query_tfhe_status.go — TFHE real-time status REST handler (v0.5.3).
//
// Endpoint: GET /zytherion/privacy/v1/tfhe/status
//
// Returns a JSON payload describing the current TFHE subsystem state:
//
//	{
//	  "enabled": true,
//	  "version": "tfhe-rs (FheUint32 / 32-bit Levelled FHE)",
//	  "erasure_coding": "12+4=16 shards (DataShards=12, ParityShards=4)",
//	  "replication_factor": 3,
//	  "node_id": "local-node",
//	  "active_commitments": 42,
//	  "shard_store_ready": true
//	}
package keeper

import (
	"encoding/json"
	"net/http"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"zytherion/x/privacy/types"
)

// TFHEStatusResponse is the JSON body returned by the TFHE status endpoint.
type TFHEStatusResponse struct {
	Enabled           bool   `json:"enabled"`
	Version           string `json:"version"`
	ErasureCoding     string `json:"erasure_coding"`
	ReplicationFactor int    `json:"replication_factor"`
	NodeID            string `json:"node_id"`
	ActiveCommitments uint64 `json:"active_commitments"`
	ShardStoreReady   bool   `json:"shard_store_ready"`
}

// TFHEStatusHTTPHandler returns an http.HandlerFunc that serves the TFHE status
// endpoint at GET /zytherion/privacy/v1/tfhe/status.
//
// The handler reads live state from the KV store (quota counters, shard store
// availability) and returns a JSON summary. TFHE is always enabled in v0.5.3+.
//
// Registration (in module.go RegisterGRPCGatewayRoutes):
//
//	mux.HandlePath("GET", "/zytherion/privacy/v1/tfhe/status",
//	    keeper.TFHEStatusHTTPHandler(k, clientCtx))
func TFHEStatusHTTPHandler(k Keeper, ctxGetter func() sdk.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Count all active commitments by iterating the TFHE quota prefix.
		var totalActive uint64
		if ctxGetter != nil {
			ctx := ctxGetter()
			totalActive = countAllActiveCommitments(k, ctx)
		}

		resp := TFHEStatusResponse{
			Enabled:           true, // TFHE is always active in v0.5.3+
			Version:           "tfhe-rs (FheUint32 / 32-bit Levelled FHE)",
			ErasureCoding:     "12+4=16 shards (DataShards=12, ParityShards=4)",
			ReplicationFactor: 3,
			NodeID:            k.nodeID,
			ActiveCommitments: totalActive,
			ShardStoreReady:   k.shardStore != nil,
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
		}
	}
}

// countAllActiveCommitments iterates the TFHE quota KV store prefix and sums
// all per-address active commitment counts. O(n) in unique submitter addresses.
func countAllActiveCommitments(k Keeper, ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	prefix := types.KeyPrefix(types.TFHEQuotaKeyPrefix)
	iter := sdk.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	var total uint64
	for ; iter.Valid(); iter.Next() {
		bz := iter.Value()
		if len(bz) == 8 {
			// Quota values are stored as big-endian uint64 (see setTFHEQuota in keeper.go).
			v := uint64(bz[0])<<56 | uint64(bz[1])<<48 | uint64(bz[2])<<40 |
				uint64(bz[3])<<32 | uint64(bz[4])<<24 | uint64(bz[5])<<16 |
				uint64(bz[6])<<8 | uint64(bz[7])
			total += v
		}
	}
	return total
}
