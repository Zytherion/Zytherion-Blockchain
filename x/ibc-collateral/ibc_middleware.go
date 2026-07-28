package ibccollateral

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channel "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"

	"zytherion/x/ibc-collateral/keeper"
	"zytherion/x/ibc-collateral/types"
)

// Ensure IBCMiddleware satisfies the IBCModule interface at compile time.
var _ porttypes.IBCModule = IBCMiddleware{}

// IBCMiddleware wraps the IBC transfer module and intercepts incoming fungible
// token packets to record when whitelisted collateral assets arrive on-chain.
type IBCMiddleware struct {
	// IBCModule is the underlying ICS-20 transfer module.
	IBCModule porttypes.IBCModule
	// keeper is the ibccollateral module keeper.
	keeper keeper.Keeper
}

// NewIBCMiddleware constructs an IBCMiddleware wrapping the given underlying module.
func NewIBCMiddleware(module porttypes.IBCModule, k keeper.Keeper) IBCMiddleware {
	return IBCMiddleware{
		IBCModule: module,
		keeper:    k,
	}
}

// ─── Channel lifecycle ─────────────────────────────────────────────────────────

// OnChanOpenInit delegates to the underlying module.
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channel.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channel.Counterparty,
	version string,
) (string, error) {
	return im.IBCModule.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
}

// OnChanOpenTry delegates to the underlying module.
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channel.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channel.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return im.IBCModule.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck delegates to the underlying module.
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return im.IBCModule.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm delegates to the underlying module.
func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.IBCModule.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit delegates to the underlying module.
func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.IBCModule.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm delegates to the underlying module.
func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.IBCModule.OnChanCloseConfirm(ctx, portID, channelID)
}

// ─── Packet lifecycle ──────────────────────────────────────────────────────────

// OnRecvPacket is called when an inbound IBC packet is received.
//
// Flow:
//  1. Delegate to the underlying transfer module so the actual ICS-20 logic
//     (minting / escrowing tokens) runs first.
//  2. If the packet carried a FungibleTokenPacketData whose denom is
//     whitelisted as collateral, emit a EventTypeCollateralReceived event so
//     off-chain listeners can react.
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channel.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	// 1. Let the underlying ICS-20 module process the transfer.
	ack := im.IBCModule.OnRecvPacket(ctx, packet, relayer)

	// 2. Only proceed if the underlying module accepted the packet.
	if !ack.Success() {
		return ack
	}

	// 3. Parse the fungible token packet data.
	var data transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		// Packet data is not ICS-20 — nothing to do for us.
		return ack
	}

	// 4. Resolve the IBC denom that the receiver will see on this chain.
	//    IBC transfer uses a path-prefixed denom such as
	//    "transfer/<channelID>/<originalDenom>" or the plain denom for native assets.
	ibcDenom := transfertypes.GetDenomPrefix(packet.GetDestPort(), packet.GetDestChannel()) + data.Denom
	if transfertypes.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.Denom) {
		// Source chain is this chain — denom is used as-is.
		ibcDenom = data.Denom
	}

	// 5. Check whether the denom is in the collateral whitelist.
	asset, err := im.keeper.GetCollateralAsset(ctx, ibcDenom)
	if err != nil {
		// Not whitelisted — nothing to do.
		return ack
	}
	if !asset.IsActive {
		return ack
	}

	// 6. Emit a collateral-received event.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCollateralReceived,
			sdk.NewAttribute(types.AttributeKeyOwner, data.Receiver),
			sdk.NewAttribute(types.AttributeKeyIBCDenom, ibcDenom),
			sdk.NewAttribute(types.AttributeKeyAmount, data.Amount),
		),
	)

	im.keeper.Logger(ctx).Info("IBC collateral received",
		"receiver", data.Receiver,
		"denom", ibcDenom,
		"amount", data.Amount,
	)

	return ack
}

// OnAcknowledgementPacket delegates to the underlying module.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channel.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return im.IBCModule.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket delegates to the underlying module.
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channel.Packet,
	relayer sdk.AccAddress,
) error {
	return im.IBCModule.OnTimeoutPacket(ctx, packet, relayer)
}
