package keeper

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DecryptBalanceResponse is the JSON body returned by the decrypt-balance endpoint.
type DecryptBalanceResponse struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Note    string `json:"note"`
}

// QueryDecryptBalance decrypts the on-chain encrypted balance for a given address.
//
// âš ï¸  PoC / Demo Only: the TFHE client key lives in the node's in-memory fhe.Context
// and is used here to decrypt. In production, only the balance owner should
// decrypt using their own off-chain client key.
func (k Keeper) QueryDecryptBalance(ctx sdk.Context, addrStr string) (DecryptBalanceResponse, error) {
	addr, err := sdk.AccAddressFromBech32(addrStr)
	if err != nil {
		return DecryptBalanceResponse{}, sdkerrors.ErrInvalidAddress.Wrapf("invalid address %q: %s", addrStr, err)
	}
	if k.fheCtx == nil {
		return DecryptBalanceResponse{}, fmt.Errorf("FHE context not initialised on this node")
	}
	if !k.HasEncryptedBalance(ctx, addr) {
		return DecryptBalanceResponse{}, fmt.Errorf("no encrypted balance for %s", addrStr)
	}
	balance, err := k.DecryptBalance(ctx, addr)
	if err != nil {
		return DecryptBalanceResponse{}, fmt.Errorf("decrypt: %w", err)
	}
	return DecryptBalanceResponse{
		Address: addrStr,
		Balance: balance,
		Note:    "[PoC] Decrypted by node using in-memory TFHE client key.",
	}, nil
}

// RegisterDecryptBalanceRoute registers the custom REST endpoint on the provided
// gorilla/mux router. Call this once from app.RegisterAPIRoutes.
//
//	GET /zytherion/privacy/v1/decrypt-balance/{address}
//
// ctxFn returns a read-only sdk.Context from the latest committed block.
func (k Keeper) RegisterDecryptBalanceRoute(router *mux.Router, ctxFn func() sdk.Context) {
	router.HandleFunc(
		"/zytherion/privacy/v1/decrypt-balance/{address}",
		func(w http.ResponseWriter, r *http.Request) {
			address := mux.Vars(r)["address"]
			ctx := ctxFn()

			resp, err := k.QueryDecryptBalance(ctx, address)

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