package types

// Event type and attribute key constants for the privacy module.
const (
	// EventTypeEncryptedTransfer is emitted by MsgEncryptedTransfer handlers.
	EventTypeEncryptedTransfer = "encrypted_transfer"

	// EventTypeDeposit is emitted by MsgDeposit handlers when a plaintext
	// coin is successfully escrowed and an encrypted balance is updated.
	EventTypeDeposit = "privacy_deposit"

	// AttributeKeySender is the event attribute key for the transfer originator.
	AttributeKeySender = "sender"

	// AttributeKeyRecipient is the event attribute key for the transfer recipient.
	AttributeKeyRecipient = "recipient"

	// AttributeKeyCreator is the event attribute key for the deposit originator.
	AttributeKeyCreator = "creator"

	// AttributeKeyDepositDenom is the event attribute key for the deposited coin denomination.
	AttributeKeyDepositDenom = "denom"
)

