// msg_tfhe_submit.go — sdk.Msg interface implementation for MsgTFHESubmit.
//
// Struct definitions (MsgTFHESubmit, MsgTFHESubmitResponse) live in tx_v3.pb.go.
// This file provides only the sdk.Msg methods and query/response helper types.
package types

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ── Query types (not in proto yet) ────────────────────────────────────────────

// QueryTFHEResultRequest is the query request for on-demand ciphertext retrieval.
type QueryTFHEResultRequest struct {
	CommitmentHash []byte `json:"commitment_hash"`
}

// QueryTFHEResultResponse returns the reconstructed TFHE ciphertext.
type QueryTFHEResultResponse struct {
	CommitmentHash    []byte `json:"commitment_hash"`
	ResultCiphertext  []byte `json:"result_ciphertext"`
	ReconstructedFrom uint32 `json:"reconstructed_from"`
}

// ── sdk.Msg interface for MsgTFHESubmit ───────────────────────────────────────

const TypeMsgTFHESubmit = "tfhe_submit"

var _ sdk.Msg = &MsgTFHESubmit{}

// NewMsgTFHESubmit constructs a MsgTFHESubmit from sender and raw ciphertext.
func NewMsgTFHESubmit(sender string, ciphertext []byte) *MsgTFHESubmit {
	return &MsgTFHESubmit{
		Sender:     sender,
		Ciphertext: ciphertext,
	}
}

func (msg *MsgTFHESubmit) Route() string { return RouterKey }
func (msg *MsgTFHESubmit) Type() string  { return TypeMsgTFHESubmit }

func (msg *MsgTFHESubmit) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgTFHESubmit) GetSignBytes() []byte {
	bz, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

func (msg *MsgTFHESubmit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAddress, err)
	}
	if len(msg.Ciphertext) == 0 {
		return errors.New("ciphertext must not be empty")
	}
	if len(msg.Ciphertext) > 32*1024 {
		return fmt.Errorf("ciphertext too large: %d bytes (max 32 KB)", len(msg.Ciphertext))
	}
	return nil
}
