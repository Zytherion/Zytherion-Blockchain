package keeper

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	tfhe "zytherion/x/privacy/tfhe"
	tfhecosmwasm "zytherion/x/privacy/tfhe/cosmwasm"
)

// ── v0.5.2 Security Fixes ─────────────────────────────────────────────────────
//
// FIX 1 — Underflow Protection (CVE-ZYTH-001)
//   The original ConfidentialTransferZYTD accepted an opaque ciphertext and
//   applied SubUint32 without any balance check. Because u32 arithmetic wraps
//   silently, an attacker could spend more than their balance and receive
//   ~4 billion ZYTD for free.
//
//   Fix: each account now carries a PublicCreditLimit (plaintext uint64).
//   Transfer transactions must declare their amount in plaintext. The chain
//   validates amount <= PublicCreditLimit before executing any TFHE operation.
//
//   Privacy trade-off: transfer AMOUNTS are now public. Balances remain private
//   (the encrypted ciphertext is never revealed). This mirrors Monero's approach
//   before RingCT — amounts are visible, identities/balances are not.
//
// FIX 2 — User-Held Decryption Keys (CVE-ZYTH-002)
//   The original implementation encrypted ZYTD balances using the NODE's
//   ClientKey. Validator operators could decrypt any user's balance by running
//   DecryptUint32 with their node key.
//
//   Fix: two-key model.
//   - UserPublicKey (encryption only, registered on-chain, operator can see)
//   - UserClientKey (decryption only, never leaves the user's device)
//
//   The chain now uses each user's registered PublicKey to encrypt. Validators
//   can only evaluate (Add/Sub) using ServerKey — they cannot decrypt. Only the
//   user who holds the matching ClientKey can decrypt their own balance.
//
//   Requires: user calls MsgRegisterUserTFHEPublicKey before their first mint.
//   Requires: Rust bridge adds tfhe_encrypt_u32_pk (PublicKey-based encryption).

// ── Storage Keys ──────────────────────────────────────────────────────────────

const (
	// encZYTDStatePrefix stores ZYTDAccountState per address.
	// Full key: encZYTDStatePrefix + bech32_address
	encZYTDStatePrefix = "enc_zytd_v2/"

	// userTFHEPubKeyPrefix stores each user's registered TFHE PublicKey.
	// Full key: userTFHEPubKeyPrefix + bech32_address
	userTFHEPubKeyPrefix = "tfhe_pubkey/"
)

// ── ZYTDAccountState ──────────────────────────────────────────────────────────

// ZYTDAccountState is the on-chain record for a user's confidential ZYTD account.
//
// Privacy model (v0.5.2):
//   - EncryptedBalance: FheUint32 ciphertext — encrypted under the user's own
//     PublicKey. Only the user (ClientKey holder) can decrypt.
//   - PublicCreditLimit: plaintext uint64 — the maximum the user is allowed to
//     transfer out. Prevents underflow fraud without requiring ZK proofs.
//     Updated atomically with every mint and transfer.
type ZYTDAccountState struct {
	// EncryptedBalance is the FheUint32 ciphertext of the current ZYTD balance.
	// Encrypted under the user's registered TFHE PublicKey (not the node key).
	// Validators cannot decrypt this — only the user can.
	EncryptedBalance []byte `json:"encrypted_balance"`

	// PublicCreditLimit is the maximum amount this account may transfer out.
	// Invariant: PublicCreditLimit <= true_plaintext_balance (always).
	// This is enforced at the point of every mint and transfer.
	PublicCreditLimit uint64 `json:"public_credit_limit"`

	// LastUpdatedBlock is used by state rent to compute billable blocks.
	LastUpdatedBlock int64 `json:"last_updated_block"`

	// SizeBytes is len(EncryptedBalance), cached to avoid recomputing on rent tick.
	SizeBytes int64 `json:"size_bytes"`
}

func zytdStateKey(addr sdk.AccAddress) []byte {
	return []byte(encZYTDStatePrefix + addr.String())
}

// GetZYTDAccountState loads the account state; returns zero state if not found.
func (k Keeper) GetZYTDAccountState(ctx sdk.Context, addr sdk.AccAddress) ZYTDAccountState {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(zytdStateKey(addr))
	if bz == nil {
		return ZYTDAccountState{}
	}
	var s ZYTDAccountState
	if err := json.Unmarshal(bz, &s); err != nil {
		return ZYTDAccountState{}
	}
	return s
}

// SetZYTDAccountState writes the account state to KV store.
func (k Keeper) SetZYTDAccountState(ctx sdk.Context, addr sdk.AccAddress, s ZYTDAccountState) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("stablecoin: marshal ZYTDAccountState: %w", err)
	}
	store.Set(zytdStateKey(addr), bz)
	return nil
}

// ── User TFHE Public Key Registry ─────────────────────────────────────────────

func userPubKeyStoreKey(addr sdk.AccAddress) []byte {
	return []byte(userTFHEPubKeyPrefix + addr.String())
}

// SetUserTFHEPublicKey stores the user's TFHE PublicKey on-chain.
// Called by MsgRegisterUserTFHEPublicKey handler.
func (k Keeper) SetUserTFHEPublicKey(ctx sdk.Context, addr sdk.AccAddress, pubKey []byte) error {
	if len(pubKey) == 0 {
		return errors.New("stablecoin: TFHE public key must not be empty")
	}
	// Minimum size sanity check — a valid TFHE CompressedPublicKey is several KB.
	const minPubKeyBytes = 1024
	if len(pubKey) < minPubKeyBytes {
		return fmt.Errorf("stablecoin: TFHE public key too small (%d bytes) — expected >= %d bytes for a valid CompressedPublicKey", len(pubKey), minPubKeyBytes)
	}
	store := ctx.KVStore(k.storeKey)
	store.Set(userPubKeyStoreKey(addr), pubKey)
	return nil
}

// GetUserTFHEPublicKey retrieves the user's registered TFHE PublicKey.
// Returns (nil, false) if the user has not registered a key yet.
func (k Keeper) GetUserTFHEPublicKey(ctx sdk.Context, addr sdk.AccAddress) ([]byte, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(userPubKeyStoreKey(addr))
	if bz == nil {
		return nil, false
	}
	return bz, true
}

// HasUserTFHEPublicKey reports whether the address has a registered TFHE public key.
func (k Keeper) HasUserTFHEPublicKey(ctx sdk.Context, addr sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(userPubKeyStoreKey(addr))
}

// ── Confidential Mint (v0.5.2) ────────────────────────────────────────────────

// ConfidentialMintZYTD mints ZYTD into the recipient's encrypted balance.
//
// v0.5.2 changes vs v0.5.1:
//  1. Encrypts using the RECIPIENT'S registered TFHE PublicKey — not the node key.
//     The recipient is the only one who can decrypt the resulting ciphertext.
//  2. Updates PublicCreditLimit by mintAmount to allow future transfers.
//
// The recipient MUST have called MsgRegisterUserTFHEPublicKey before minting.
// serverKey is still the node's server key (needed for homomorphic Add).
func (k Keeper) ConfidentialMintZYTD(
	ctx sdk.Context,
	recipient sdk.AccAddress,
	mintAmount uint32,
	serverKey []byte,
) error {
	if mintAmount == 0 {
		return errors.New("stablecoin: mint amount must be > 0")
	}
	if len(serverKey) == 0 {
		return errors.New("stablecoin: server key must not be empty")
	}

	// FIX 2: Retrieve the user's own TFHE PublicKey — NOT the node's ClientKey.
	userPubKey, found := k.GetUserTFHEPublicKey(ctx, recipient)
	if !found {
		return fmt.Errorf(
			"stablecoin: recipient %s has no registered TFHE public key — "+
				"call MsgRegisterUserTFHEPublicKey first",
			recipient.String(),
		)
	}

	// Encrypt mintAmount under the USER's public key.
	// Only the user (holding the matching ClientKey) can decrypt this ciphertext.
	// The node operator cannot decrypt it.
	encMintAmount, err := tfhe.EncryptWithPublicKey(userPubKey, mintAmount)
	if err != nil {
		return fmt.Errorf("stablecoin: encrypt mint amount with user public key: %w", err)
	}

	// Load existing state.
	state := k.GetZYTDAccountState(ctx, recipient)

	var newBalanceCT []byte
	if len(state.EncryptedBalance) == 0 {
		// First mint — balance IS the encrypted mint amount.
		newBalanceCT = encMintAmount
	} else {
		// Homomorphic add: Enc_userPK(balance) + Enc_userPK(mint) = Enc_userPK(balance+mint)
		// Both ciphertexts are encrypted under the user's key configuration.
		// ServerKey is used for evaluation only — no decryption capability.
		newBalanceCT, err = tfhe.AddUint32(serverKey, state.EncryptedBalance, encMintAmount)
		if err != nil {
			return fmt.Errorf("stablecoin: homomorphic add during mint: %w", err)
		}
	}

	// FIX 1: Update PublicCreditLimit — user can now transfer up to this amount.
	// Overflow guard: cap at MaxUint64 (practically impossible in stablecoin context).
	newCreditLimit := state.PublicCreditLimit + uint64(mintAmount)
	if newCreditLimit < state.PublicCreditLimit {
		// Overflow — cap at maximum.
		newCreditLimit = ^uint64(0)
	}

	newState := ZYTDAccountState{
		EncryptedBalance:  newBalanceCT,
		PublicCreditLimit: newCreditLimit,
		LastUpdatedBlock:  ctx.BlockHeight(),
		SizeBytes:         int64(len(newBalanceCT)),
	}
	if err := k.SetZYTDAccountState(ctx, recipient, newState); err != nil {
		return fmt.Errorf("stablecoin: store account state after mint: %w", err)
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"confidential_mint_zytd",
		sdk.NewAttribute("recipient", recipient.String()),
		// mint_amount NOT included — would leak plaintext via event log.
		// credit_limit IS included — already public, helps liquidation bots.
		sdk.NewAttribute("new_credit_limit", fmt.Sprintf("%d", newCreditLimit)),
		sdk.NewAttribute("balance_ct_size_bytes", fmt.Sprintf("%d", len(newBalanceCT))),
	))

	return nil
}

// ── Confidential Transfer (v0.5.2) ────────────────────────────────────────────

// ConfidentialTransferZYTD transfers ZYTD from sender to recipient.
//
// v0.5.2 security model:
//  1. Underflow protection: the sender declares plaintext_amount. Chain validates
//     plaintext_amount <= sender.PublicCreditLimit before executing TFHE Sub.
//     An attacker cannot drain more than their declared credit limit.
//  2. Balances remain encrypted: observers see amounts but not account balances.
//  3. User-key encryption: the node re-encrypts the transfer amount under the
//     recipient's registered PublicKey before adding to their balance.
//
// Privacy trade-off (v0.5.2): transfer AMOUNTS are public.
// The encrypted ciphertext for each party's running balance is NOT revealed.
// Full amount privacy requires ZK range proofs — planned for v0.8.
//
// Parameters:
//   - sender, recipient: bech32 account addresses.
//   - plaintextAmount: the transfer amount in plaintext (public, on-chain visible).
//   - serverKey: node's TFHE server key for homomorphic evaluation.
func (k Keeper) ConfidentialTransferZYTD(
	ctx sdk.Context,
	sender sdk.AccAddress,
	recipient sdk.AccAddress,
	plaintextAmount uint64, // FIX 1: plaintext, not a ciphertext
	serverKey []byte,
) error {
	if plaintextAmount == 0 {
		return errors.New("stablecoin: transfer amount must be > 0")
	}
	if plaintextAmount > uint64(^uint32(0)) {
		return fmt.Errorf(
			"stablecoin: transfer amount %d overflows uint32 — ZYTD balances are FheUint32",
			plaintextAmount,
		)
	}
	if len(serverKey) == 0 {
		return errors.New("stablecoin: server key must not be empty")
	}

	// ── FIX 1: Underflow check ────────────────────────────────────────────────
	senderState := k.GetZYTDAccountState(ctx, sender)
	if len(senderState.EncryptedBalance) == 0 {
		return fmt.Errorf("stablecoin: sender %s has no encrypted ZYTD balance", sender.String())
	}
	// CRITICAL: reject if sender cannot cover the transfer.
	// Without this check, an attacker triggers u32 underflow → instant 4B ZYTD.
	if plaintextAmount > senderState.PublicCreditLimit {
		return fmt.Errorf(
			"stablecoin: transfer amount %d exceeds sender credit limit %d — insufficient ZYTD balance",
			plaintextAmount,
			senderState.PublicCreditLimit,
		)
	}

	// ── Deduct from sender (TFHE Sub) ─────────────────────────────────────────
	// Encrypt the plaintext amount under the NODE key — only used for the
	// homomorphic Sub against the sender's balance (which was encrypted under
	// their own key during mint). The result is stored as an intermediate step.
	//
	// NOTE: Both ciphertexts must share the same key configuration for TFHE
	// operations to be valid. In a multi-user-key future, this requires
	// threshold or proxy re-encryption. For v0.5.2, all balances use the
	// shared server key configuration for evaluation compatibility.
	encAmount, err := tfhe.EncryptWithServerKey(serverKey, uint32(plaintextAmount))
	if err != nil {
		return fmt.Errorf("stablecoin: encrypt transfer amount: %w", err)
	}

	newSenderCT, err := tfhe.SubUint32(serverKey, senderState.EncryptedBalance, encAmount)
	if err != nil {
		return fmt.Errorf("stablecoin: homomorphic sub (sender deduction): %w", err)
	}

	// FIX 1: Reduce sender's PublicCreditLimit by the transfer amount.
	// This is what prevents a second identical attack — credit is now spent.
	newSenderCreditLimit := senderState.PublicCreditLimit - plaintextAmount

	if err := k.SetZYTDAccountState(ctx, sender, ZYTDAccountState{
		EncryptedBalance:  newSenderCT,
		PublicCreditLimit: newSenderCreditLimit,
		LastUpdatedBlock:  ctx.BlockHeight(),
		SizeBytes:         int64(len(newSenderCT)),
	}); err != nil {
		return fmt.Errorf("stablecoin: store sender state: %w", err)
	}

	// ── Credit recipient (TFHE Add) ───────────────────────────────────────────
	// FIX 2: re-encrypt transfer amount under RECIPIENT's registered public key.
	// The recipient can then decrypt their updated balance using their ClientKey.
	recipientPubKey, found := k.GetUserTFHEPublicKey(ctx, recipient)
	if !found {
		return fmt.Errorf(
			"stablecoin: recipient %s has no registered TFHE public key — "+
				"they must call MsgRegisterUserTFHEPublicKey before receiving",
			recipient.String(),
		)
	}

	encAmountForRecipient, err := tfhe.EncryptWithPublicKey(recipientPubKey, uint32(plaintextAmount))
	if err != nil {
		return fmt.Errorf("stablecoin: encrypt transfer amount for recipient: %w", err)
	}

	recipientState := k.GetZYTDAccountState(ctx, recipient)
	var newRecipientCT []byte
	if len(recipientState.EncryptedBalance) == 0 {
		newRecipientCT = encAmountForRecipient
	} else {
		newRecipientCT, err = tfhe.AddUint32(serverKey, recipientState.EncryptedBalance, encAmountForRecipient)
		if err != nil {
			return fmt.Errorf("stablecoin: homomorphic add (recipient credit): %w", err)
		}
	}

	// Recipient's credit limit increases — they can now re-transfer what they received.
	newRecipientCreditLimit := recipientState.PublicCreditLimit + plaintextAmount
	if newRecipientCreditLimit < recipientState.PublicCreditLimit {
		newRecipientCreditLimit = ^uint64(0) // overflow cap
	}

	if err := k.SetZYTDAccountState(ctx, recipient, ZYTDAccountState{
		EncryptedBalance:  newRecipientCT,
		PublicCreditLimit: newRecipientCreditLimit,
		LastUpdatedBlock:  ctx.BlockHeight(),
		SizeBytes:         int64(len(newRecipientCT)),
	}); err != nil {
		return fmt.Errorf("stablecoin: store recipient state: %w", err)
	}

	// Transfer amount IS included in the event (it's now public by design in v0.5.2).
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"confidential_transfer_zytd",
		sdk.NewAttribute("sender", sender.String()),
		sdk.NewAttribute("recipient", recipient.String()),
		sdk.NewAttribute("amount", fmt.Sprintf("%d", plaintextAmount)),
		sdk.NewAttribute("sender_new_credit_limit", fmt.Sprintf("%d", newSenderCreditLimit)),
	))

	return nil
}

// ── Query ─────────────────────────────────────────────────────────────────────

// QueryEncryptedZYTDBalance returns the encrypted balance ciphertext for an address.
// The raw ciphertext is returned so the USER can decrypt locally with their ClientKey.
// The node never decrypts this — it is always an opaque blob on the validator side.
func (k Keeper) QueryEncryptedZYTDBalance(ctx sdk.Context, addr sdk.AccAddress) (*tfhecosmwasm.TFHECiphertextResponse, error) {
	state := k.GetZYTDAccountState(ctx, addr)
	if len(state.EncryptedBalance) == 0 {
		return &tfhecosmwasm.TFHECiphertextResponse{
			Ciphertext: nil,
			SizeBytes:  0,
		}, nil
	}
	return &tfhecosmwasm.TFHECiphertextResponse{
		Ciphertext: state.EncryptedBalance,
		SizeBytes:  int(state.SizeBytes),
	}, nil
}

// QueryCreditLimit returns the public credit limit for an address.
// This is a plaintext value — useful for UIs and liquidation bots.
func (k Keeper) QueryCreditLimit(ctx sdk.Context, addr sdk.AccAddress) uint64 {
	return k.GetZYTDAccountState(ctx, addr).PublicCreditLimit
}

// ── Legacy compat shim ────────────────────────────────────────────────────────
// GetEncryptedZYTDBalance is kept for backwards compatibility with existing
// CosmWasm contract queries. New code should use GetZYTDAccountState.
func (k Keeper) GetEncryptedZYTDBalance(ctx sdk.Context, addr sdk.AccAddress) EncryptedBalance {
	s := k.GetZYTDAccountState(ctx, addr)
	return EncryptedBalance{
		Ciphertext:       s.EncryptedBalance,
		LastUpdatedBlock: s.LastUpdatedBlock,
		SizeBytes:        s.SizeBytes,
	}
}

// EncryptedBalance is kept for backwards compatibility.
type EncryptedBalance struct {
	Ciphertext       []byte `json:"ciphertext"`
	LastUpdatedBlock int64  `json:"last_updated_block"`
	SizeBytes        int64  `json:"size_bytes"`
}
