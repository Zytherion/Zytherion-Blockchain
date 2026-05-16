package types

// Event type and attribute key constants for the privacy module.
const (
	// EventTypeZKTransfer is emitted by MsgZKTransfer handlers.
	EventTypeZKTransfer = "zk_transfer"

	// EventTypeInitCommitment is emitted by MsgInitCommitment handlers when
	// coins are escrowed and a commitment is successfully registered.
	EventTypeInitCommitment = "init_commitment"

	// EventTypeEncryptedTransfer is retained for backward-compat with indexers.
	// Deprecated: use EventTypeZKTransfer.
	EventTypeEncryptedTransfer = "encrypted_transfer"

	// EventTypeDeposit is retained for backward-compat with indexers.
	// Deprecated: use EventTypeInitCommitment.
	EventTypeDeposit = "privacy_deposit"

	// AttributeKeySender is the event attribute key for the transfer originator.
	AttributeKeySender = "sender"

	// AttributeKeyRecipient is the event attribute key for the transfer recipient.
	AttributeKeyRecipient = "recipient"

	// AttributeKeyCreator is the event attribute key for the deposit/commitment originator.
	AttributeKeyCreator = "creator"

	// AttributeKeyDepositDenom is the event attribute key for the deposited coin denomination.
	AttributeKeyDepositDenom = "denom"
)


