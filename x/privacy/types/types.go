package types

// Event type constants for the privacy module — supplemental to keys.go.
// Kept here for any indexers that subscribed to the old event names.
const (
	// EventTypeEncryptedTransfer — legacy event name, no longer emitted.
	EventTypeEncryptedTransfer = "encrypted_transfer"

	// EventTypeDeposit — legacy event name, no longer emitted.
	EventTypeDeposit = "privacy_deposit"
)
