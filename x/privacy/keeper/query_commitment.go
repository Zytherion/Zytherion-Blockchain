// query_commitment.go — REST endpoint for querying an account's ZK commitment.
//
// Returns the raw 32-byte commitment stored on-chain for a given address.
// The commitment is a MiMC hash — the user decrypts it off-chain using
// their own blinding factor. No plaintext amounts are ever returned by
// the node (it doesn't have them).
//
// Endpoint:
//
//	GET /zytherion/privacy/v1/commitment/{address}
package keeper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CommitmentResponse is the JSON body returned by the commitment endpoint.
type CommitmentResponse struct {
	// Address is the bech32 address queried.
	Address string `json:"address"`
	// CommitmentHex is the hex-encoded 32-byte on-chain commitment.
	// Decode off-chain with your blinding factor to verify your balance.
	CommitmentHex string `json:"commitment_hex"`
	// Note provides context for the caller.
	Note string `json:"note"`
}

// QueryCommitment returns the raw commitment for a given address.
func (k Keeper) QueryCommitment(ctx sdk.Context, addrStr string) (CommitmentResponse, error) {
	addr, err := sdk.AccAddressFromBech32(addrStr)
	if err != nil {
		return CommitmentResponse{}, sdkerrors.ErrInvalidAddress.Wrapf("invalid address %q: %s", addrStr, err)
	}
	commitment, found := k.GetCommitment(ctx, addr)
	if !found {
		return CommitmentResponse{}, fmt.Errorf("no commitment registered for %s", addrStr)
	}
	return CommitmentResponse{
		Address:       addrStr,
		CommitmentHex: hex.EncodeToString(commitment),
		Note:          "Commitment is a MiMC hash. Verify off-chain with your blinding factor.",
	}, nil
}

// RegisterCommitmentRoute registers the GET /zytherion/privacy/v1/commitment/{address}
// REST endpoint on the provided gorilla/mux router.
func (k Keeper) RegisterCommitmentRoute(router *mux.Router, ctxFn func() sdk.Context) {
	router.HandleFunc(
		"/zytherion/privacy/v1/commitment/{address}",
		func(w http.ResponseWriter, r *http.Request) {
			address := mux.Vars(r)["address"]
			ctx := ctxFn()

			resp, err := k.QueryCommitment(ctx, address)

			w.Header().Set("Content-Type", "application/json")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   err.Error(),
					"address": address,
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		},
	).Methods(http.MethodGet)
}
