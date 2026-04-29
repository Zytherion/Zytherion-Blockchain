package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed by the privacy module.
// It covers both balance queries (used by existing code) and coin transfers
// (used by MsgDeposit to escrow tokens into the privacy module account).
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins

	// SendCoinsFromAccountToModule moves coins from a user account to a named
	// module account (escrow).  Used by the Deposit handler to lock the
	// plaintext tokens before issuing an encrypted balance.
	SendCoinsFromAccountToModule(
		ctx sdk.Context,
		senderAddr sdk.AccAddress,
		recipientModule string,
		amt sdk.Coins,
	) error
}

