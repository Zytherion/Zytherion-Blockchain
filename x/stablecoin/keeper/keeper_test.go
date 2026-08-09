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

	ibccollateraltypes "zytherion/x/ibc-collateral/types"
	oracletypes "zytherion/x/oracle/types"
	"zytherion/x/stablecoin/keeper"
	"zytherion/x/stablecoin/types"
)

type MockOracleKeeper struct {
	mock.Mock
}

func (m *MockOracleKeeper) GetTWAP(ctx sdk.Context, denom string) (oracletypes.TWAPData, error) {
	args := m.Called(ctx, denom)
	return args.Get(0).(oracletypes.TWAPData), args.Error(1)
}

func (m *MockOracleKeeper) GetLatestPrice(ctx sdk.Context, denom string) (oracletypes.PriceEntry, error) {
	args := m.Called(ctx, denom)
	return args.Get(0).(oracletypes.PriceEntry), args.Error(1)
}

type MockIBCCollateralKeeper struct {
	mock.Mock
}

func (m *MockIBCCollateralKeeper) GetCollateralAsset(ctx sdk.Context, ibcDenom string) (ibccollateraltypes.CollateralAsset, error) {
	args := m.Called(ctx, ibcDenom)
	return args.Get(0).(ibccollateraltypes.CollateralAsset), args.Error(1)
}

func (m *MockIBCCollateralKeeper) GetPosition(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string) (ibccollateraltypes.CollateralPosition, error) {
	args := m.Called(ctx, owner, ibcDenom)
	return args.Get(0).(ibccollateraltypes.CollateralPosition), args.Error(1)
}

func (m *MockIBCCollateralKeeper) SetPosition(ctx sdk.Context, pos ibccollateraltypes.CollateralPosition) {
	m.Called(ctx, pos)
}

func (m *MockIBCCollateralKeeper) LockCollateral(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, amount sdk.Int) error {
	args := m.Called(ctx, owner, ibcDenom, amount)
	return args.Error(0)
}

func (m *MockIBCCollateralKeeper) UnlockCollateral(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, amount sdk.Int) error {
	args := m.Called(ctx, owner, ibcDenom, amount)
	return args.Error(0)
}

func (m *MockIBCCollateralKeeper) UpdateMintedZYTD(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, delta sdk.Int) error {
	args := m.Called(ctx, owner, ibcDenom, delta)
	return args.Error(0)
}

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

func (m *MockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	args := m.Called(ctx, moduleName, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	args := m.Called(ctx, moduleName, amt)
	return args.Error(0)
}

func (m *MockBankKeeper) SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	args := m.Called(ctx, addr)
	return args.Get(0).(sdk.Coins)
}

func (m *MockBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	args := m.Called(ctx, addr, denom)
	return args.Get(0).(sdk.Coin)
}

func TestStablecoinMinting(t *testing.T) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockOracle := new(MockOracleKeeper)
	mockIBC := new(MockIBCCollateralKeeper)
	mockBank := new(MockBankKeeper)

	k := keeper.NewKeeper(cdc, storeKey, mockBank, mockOracle, mockIBC)
	ctx := sdk.NewContext(stateStore, tmproto.Header{Height: 1}, false, log.NewNopLogger())

	// Test SetParams
	params := types.DefaultStablecoinParams()
	k.SetParams(ctx, params)
	require.Equal(t, params, k.GetParams(ctx))

	sender := sdk.AccAddress([]byte("sender_addr"))
	collateralDenom := "ibc/axlUSDC"
	collateralAmount := sdk.NewInt(1000)
	requestedZYTD := sdk.NewInt(500)

	// Mock queries and calls
	mockOracle.On("GetTWAP", ctx, collateralDenom).Return(oracletypes.TWAPData{
		Denom: collateralDenom,
		TWAP:  sdk.MustNewDecFromStr("1.00"),
	}, nil)

	mockIBC.On("GetCollateralAsset", ctx, collateralDenom).Return(ibccollateraltypes.CollateralAsset{
		IBCDenom:  collateralDenom,
		IsActive:  true,
		MinRatio:  sdk.MustNewDecFromStr("1.10"),
	}, nil)

	mockIBC.On("LockCollateral", ctx, sender, collateralDenom, collateralAmount).Return(nil)
	mockBank.On("MintCoins", ctx, types.ModuleAccountName, sdk.NewCoins(sdk.NewCoin(types.ZYTDDenom, requestedZYTD))).Return(nil)
	mockBank.On("SendCoinsFromModuleToAccount", ctx, types.ModuleAccountName, sender, sdk.NewCoins(sdk.NewCoin(types.ZYTDDenom, requestedZYTD))).Return(nil)
	mockIBC.On("UpdateMintedZYTD", ctx, sender, collateralDenom, requestedZYTD).Return(nil)

	// Call MsgServer Mint
	msgSrv := keeper.NewMsgServerImpl(k)
	req := types.NewMsgMintZYTD(sender.String(), collateralDenom, collateralAmount, requestedZYTD)
	res, err := msgSrv.MintZYTD(sdk.WrapSDKContext(ctx), req)
	require.NoError(t, err)
	require.Equal(t, requestedZYTD, res.MintedAmount)
}
