package keeper_test

import (
	"testing"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"zytherion/x/ibc-collateral/keeper"
	"zytherion/x/ibc-collateral/types"
)

// MockBankKeeper mock bank keeper for transfer testing.
type MockBankKeeper struct {
	mock.Mock
}

func (m *MockBankKeeper) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	args := m.Called(ctx, fromAddr, toAddr, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	args := m.Called(ctx, senderAddr, recipientModule, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	args := m.Called(ctx, senderModule, recipientAddr, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	args := m.Called(ctx, moduleName, amounts)
	return args.Error(0)
}

func (m *MockBankKeeper) BurnCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	args := m.Called(ctx, moduleName, amounts)
	return args.Error(0)
}

func (m *MockBankKeeper) SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	args := m.Called(ctx, addr)
	return args.Get(0).(sdk.Coins)
}

type MockAccountKeeper struct {
	mock.Mock
}

func (m *MockAccountKeeper) GetModuleAddress(name string) sdk.AccAddress {
	args := m.Called(name)
	return args.Get(0).(sdk.AccAddress)
}

func TestIBCCollateralKeeper(t *testing.T) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockBank := new(MockBankKeeper)
	mockAccount := new(MockAccountKeeper)

	k := keeper.NewKeeper(cdc, storeKey, mockBank, mockAccount)
	ctx := sdk.NewContext(stateStore, tmproto.Header{Height: 1}, false, log.NewNopLogger())

	// Test Register Asset
	asset := types.CollateralAsset{
		IBCDenom:             "ibc/axlUSDC",
		BaseDenom:            "axlUSDC",
		MinRatio:             sdk.MustNewDecFromStr("1.10"),
		LiquidationThreshold: sdk.MustNewDecFromStr("1.05"),
		IsActive:             true,
	}

	err := k.RegisterCollateralAsset(ctx, asset)
	require.NoError(t, err)

	res, err := k.GetCollateralAsset(ctx, "ibc/axlUSDC")
	require.NoError(t, err)
	require.Equal(t, asset.BaseDenom, res.BaseDenom)

	// Test LockCollateral
	owner := sdk.AccAddress([]byte("owner_addr"))
	vaultAddr := sdk.AccAddress([]byte("vault_addr"))
	amount := sdk.NewInt(1000)

	mockBank.On("SendCoinsFromAccountToModule", ctx, owner, types.ModuleAccountName, sdk.NewCoins(sdk.NewCoin("ibc/axlUSDC", amount))).Return(nil)
	mockAccount.On("GetModuleAddress", types.ModuleAccountName).Return(vaultAddr)

	err = k.LockCollateral(ctx, owner, "ibc/axlUSDC", amount)
	require.NoError(t, err)

	pos, err := k.GetPosition(ctx, owner, "ibc/axlUSDC")
	require.NoError(t, err)
	require.Equal(t, amount, pos.Amount)

	require.Equal(t, amount, k.GetTotalLocked(ctx, "ibc/axlUSDC"))
}
