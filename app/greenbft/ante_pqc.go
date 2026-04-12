package greenbft

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"zytherion/x/privacy/pqc"
)

// PQCAnteDecorator is an SDK AnteDecorator that computes and logs a
// SHA3-256 PQC block-hash commitment for each incoming transaction.
//
// The hash is computed via pqc.GenerateBlockHash over the serialised
// transaction bytes. This provides a quantum-resistant, domain-separated
// fingerprint that can later be used for PQC inclusion proofs.
//
// The decorator is placed FIRST in the ante chain to keep the PQC hash
// available to all downstream decorators via the logger.
type PQCAnteDecorator struct{}

// NewPQCAnteDecorator constructs a PQCAnteDecorator ready to chain.
func NewPQCAnteDecorator() PQCAnteDecorator {
	return PQCAnteDecorator{}
}

// AnteHandle implements sdk.AnteDecorator.
// It computes the SHA3-256 PQC hash of the serialised transaction bytes and
// records it on the context logger for block explorers and auditors.
// On production chains a future upgrade can persist this hash, enabling
// zero-knowledge PQC inclusion proofs.
func (d PQCAnteDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	// Encode the tx to bytes for hashing.
	// sdk.Tx does not expose raw bytes directly; we marshal the messages
	// to a canonical representation using the sdk codec.
	txBytes := marshalTxMsgs(tx)

	// GenerateBlockHash with Height=0 produces a standalone SHA3-256
	// commitment over the tx bytes (not a full block hash).
	pqcHash := pqc.GenerateBlockHash(pqc.BlockHashInput{
		Height:       0,
		Transactions: [][]byte{txBytes},
	})

	if !simulate {
		ctx.Logger().Debug("greenbft PQC ante",
			"pqc_tx_hash", pqcHashHex(pqcHash),
			"tx_msg_count", len(tx.GetMsgs()),
		)
	}

	return next(ctx, tx, simulate)
}

// marshalTxMsgs produces a deterministic byte representation of the tx
// suitable for PQC hashing. We concatenate the type URLs and raw values of
// all messages — a stable, codec-independent representation.
func marshalTxMsgs(tx sdk.Tx) []byte {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return []byte{}
	}
	var buf []byte
	for _, msg := range msgs {
		typeURL := sdk.MsgTypeURL(msg)
		buf = append(buf, []byte(typeURL)...)
		// Append encoded message bytes if the msg implements proto.Marshaler.
		if pm, ok := msg.(interface{ Marshal() ([]byte, error) }); ok {
			b, err := pm.Marshal()
			if err == nil {
				buf = append(buf, b...)
			}
		}
	}
	return buf
}

// pqcHashHex returns the hex-encoded PQC hash for logging (first 8 bytes).
func pqcHashHex(h []byte) string {
	const hextable = "0123456789abcdef"
	n := 8
	if len(h) < n {
		n = len(h)
	}
	dst := make([]byte, n*2)
	for i := 0; i < n; i++ {
		dst[i*2] = hextable[h[i]>>4]
		dst[i*2+1] = hextable[h[i]&0x0f]
	}
	return string(dst)
}
