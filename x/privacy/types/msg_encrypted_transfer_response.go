package types

// MsgEncryptedTransferResponse is returned by the EncryptedTransfer handler
// upon success.  It carries no plaintext data — the updated ciphertext is
// stored on-chain and retrievable via the encrypted balance query.
type MsgEncryptedTransferResponse struct{}

// ProtoMessage implements proto.Message (no-op stub for non-proto types).
func (r *MsgEncryptedTransferResponse) ProtoMessage() {}

// Reset is a no-op stub required by proto.Message.
func (r *MsgEncryptedTransferResponse) Reset() {}

// String returns a human-readable form.
func (r *MsgEncryptedTransferResponse) String() string { return "MsgEncryptedTransferResponse{}" }
