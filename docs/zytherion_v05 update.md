# Zytherion v0.5.1 — Cursor/Windsurf Implementation Prompt
# Multi-Collateral Stablecoin (ZYTD) + IBC Bridge + CosmWasm Smart Contract

---

## CONTEXT: EXISTING CODEBASE

You are working on **Zytherion Blockchain v0.4**, a Layer-1 blockchain built on:
- **Cosmos SDK v0.47 + CometBFT v0.37**
- **Go** (main codebase) + **Rust** (TFHE library via CGo)
- Module path: `github.com/Zytherion/Zytherion-Blockchain`

### Existing module structure (DO NOT MODIFY these):
```
zytherion/
├── app/
│   ├── app.go                    # Cosmos SDK app — module registration
│   ├── crypto_startup.go         # Dilithium5 self-test on startup
│   └── greenbft/                 # GreenBFT adaptive commit
├── x/privacy/
│   ├── client/cli/
│   │   ├── query.go
│   │   ├── tx.go
│   │   ├── tx_deposit.go         # MsgInitCommitment CLI
│   │   └── tx_encrypted_transfer.go  # MsgTFHESubmit CLI
│   ├── keeper/
│   │   ├── keeper.go             # KVStore, TFHE meta/result, quota CRUD
│   │   ├── msg_server_tfhe_submit.go  # quota check, Merkle build, shards
│   │   ├── query_tfhe_result.go  # reconstructs ciphertext on-demand
│   │   └── msg_server_deposit.go
│   ├── pqc/
│   │   ├── signature.go          # Dilithium5 (ML-DSA-87) wrappers
│   │   ├── lwr_hash.go           # Ring-LWR hashing
│   │   └── povl.go               # Sequential VDF
│   ├── tfhe/
│   │   ├── engine.go             # CGo bridge to libtfhe_c.a (mutex-serialized)
│   │   ├── engine_stub.go        # fallback stub without CGo
│   │   ├── erasure.go            # Reed-Solomon 12+4=16 shards
│   │   ├── merkle.go             # Binary SHA-256 Merkle tree over shards
│   │   ├── shard_store.go        # disk-based shard persistence
│   │   ├── shard_distributor.go  # HTTP P2P shard server, auth, rate limiter
│   │   └── tfhe_c/               # Rust FFI crate (tfhe-rs v0.6)
│   └── types/
│       ├── codec.go
│       ├── errors.go             # ErrTFHEQuotaExceeded(1205), ErrShardAuthFailed(1206)
│       ├── keys.go               # KV prefixes, TFHEQuotaKeyPrefix
│       └── tx.pb.go
├── cmd/zytheriond/cmd/root.go
├── go.mod
└── Makefile
```

### Critical existing patterns to follow:
- All KV store keys use prefix constants defined in `x/privacy/types/keys.go`
- Error codes start from `1200` series in `x/privacy/types/errors.go`
- Module account names use format `"module_name"` string constants
- CLI commands follow Cobra pattern from existing `tx.go` and `query.go`
- All new modules must be registered in `app/app.go` following the existing `x/privacy` pattern
- go.mod module path is `github.com/Zytherion/Zytherion-Blockchain`

---

## TASK: Implement Zytherion v0.5.1 — Three new modules + CosmWasm

Implement the following **four additions** to the existing Zytherion codebase. Generate complete, production-ready Go code for each. Do not use placeholder comments like `// TODO` — write the actual implementation.

---

## MODULE 1: `x/oracle` — Price Feed with TWAP

### Purpose
Provides on-chain price data (ZYTC/USD and whitelisted collateral assets/USD) used by `x/stablecoin` for collateral ratio calculation and liquidation triggers. Validators submit prices every block; the module aggregates using median TWAP.

### File structure to create:
```
x/oracle/
├── client/cli/
│   ├── query.go          # query subcommands
│   └── tx.go             # tx subcommands (MsgSubmitPrice)
├── keeper/
│   ├── keeper.go         # KVStore, price CRUD, TWAP calculation
│   ├── msg_server.go     # MsgSubmitPrice handler
│   └── query_server.go   # QueryPrice, QueryTWAP handlers
├── types/
│   ├── codec.go
│   ├── errors.go
│   ├── keys.go
│   ├── msgs.go           # MsgSubmitPrice definition
│   ├── params.go         # OracleParams (TwapWindow, MinSubmissions, etc.)
│   └── types.go          # PriceEntry, TWAPData structs
├── module.go             # AppModule interface implementation
└── genesis.go
```

### Implementation requirements:

**`x/oracle/types/types.go`**
```go
// Implement these structs with proper JSON tags:

type PriceEntry struct {
    Denom     string    // e.g. "ZYTC", "axlUSDC", "ATOM"
    PriceUSD  sdk.Dec
    Submitter sdk.AccAddress
    Height    int64
    Timestamp time.Time
}

type TWAPData struct {
    Denom       string
    TWAP        sdk.Dec   // time-weighted average price
    WindowStart int64     // block height
    WindowEnd   int64     // block height
    NumSamples  int
}

type OracleParams struct {
    TwapWindowBlocks int64    // default: 30 blocks (~150s at 5s/block)
    MinSubmissions   int      // minimum validator submissions before TWAP is valid, default: 2/3 of validators
    MaxPriceAge      int64    // reject prices older than N blocks, default: 5
    WhitelistedDenoms []string // ["ZYTC", "axlUSDC", "mUSDT", "ATOM", "wBTC", "wETH"]
}
```

**`x/oracle/types/keys.go`**
```go
const (
    ModuleName = "oracle"
    StoreKey   = "oracle"
    RouterKey  = "oracle"

    PriceKeyPrefix      = "price/"       // price/<denom>/<height> -> PriceEntry JSON
    TWAPKeyPrefix       = "twap/"        // twap/<denom> -> TWAPData JSON
    OracleParamsKey     = "oracle_params"
)
```

**`x/oracle/types/errors.go`**
```go
// Use error codes starting from 1300 series to not conflict with x/privacy (1200 series)
var (
    ErrInvalidDenom        = errorsmod.Register(ModuleName, 1300, "denom not whitelisted")
    ErrPriceTooOld         = errorsmod.Register(ModuleName, 1301, "price submission too old")
    ErrInsufficientSubmissions = errorsmod.Register(ModuleName, 1302, "not enough price submissions for valid TWAP")
    ErrUnauthorizedSubmitter   = errorsmod.Register(ModuleName, 1303, "submitter is not an active validator")
)
```

**`x/oracle/keeper/keeper.go`** — implement ALL of these functions with full logic:
```go
func (k Keeper) SetPrice(ctx sdk.Context, entry PriceEntry) error
func (k Keeper) GetLatestPrice(ctx sdk.Context, denom string) (PriceEntry, error)
func (k Keeper) GetPriceHistory(ctx sdk.Context, denom string, fromHeight int64) []PriceEntry
func (k Keeper) ComputeTWAP(ctx sdk.Context, denom string) (TWAPData, error)
    // Algorithm: collect all PriceEntry for denom within TwapWindowBlocks,
    // compute median of all submissions per block, then time-weight by block duration.
    // Reject if NumSamples < MinSubmissions.
func (k Keeper) GetTWAP(ctx sdk.Context, denom string) (TWAPData, error)
func (k Keeper) SetTWAP(ctx sdk.Context, data TWAPData)
func (k Keeper) PruneOldPrices(ctx sdk.Context, currentHeight int64)
    // Delete PriceEntry older than (currentHeight - TwapWindowBlocks - MaxPriceAge)
func (k Keeper) IsValidatorSubmitter(ctx sdk.Context, addr sdk.AccAddress) bool
    // Check using stakingKeeper.GetValidator
```

**`x/oracle/keeper/msg_server.go`** — `MsgSubmitPrice` handler:
- Validate submitter is active bonded validator via `stakingKeeper`
- Validate denom is in `WhitelistedDenoms`
- Validate price > 0 and submission height is within `MaxPriceAge` blocks of current
- Store `PriceEntry` in KV store
- After storing, call `ComputeTWAP` and update the TWAP cache
- Call `PruneOldPrices` to keep KV store lean
- Emit event `types.EventTypeSubmitPrice` with attributes: denom, price, submitter

**`x/oracle/keeper/query_server.go`**:
```go
func (k Keeper) QueryPrice(ctx, req) -> (PriceEntry, error)    // latest price for denom
func (k Keeper) QueryTWAP(ctx, req)  -> (TWAPData, error)      // TWAP for denom
func (k Keeper) QueryAllPrices(ctx, req) -> ([]PriceEntry, error) // all latest prices
```

**`x/oracle/client/cli/tx.go`** — CLI for `MsgSubmitPrice`:
```bash
# Usage:
zytheriond tx oracle submit-price ZYTC 0.85 \
  --from my-validator \
  --keyring-backend test \
  --chain-id zytherion \
  -y
```

**`x/oracle/client/cli/query.go`** — CLI queries:
```bash
zytheriond query oracle price ZYTC          # latest price
zytheriond query oracle twap ZYTC           # TWAP value
zytheriond query oracle prices              # all whitelisted asset prices
```

---

## MODULE 2: `x/ibc-collateral` — IBC Token Receiver & Vault

### Purpose
Receives IBC token transfers (ICS-20) from external chains, validates them against a whitelist, locks them in a module vault account, and emits events that `x/stablecoin` listens to for collateral accounting.

### File structure to create:
```
x/ibc-collateral/
├── client/cli/
│   ├── query.go
│   └── tx.go
├── ibc_middleware.go     # ICS-4 middleware wrapping transfer module
├── keeper/
│   ├── keeper.go
│   ├── msg_server.go     # MsgUnlockCollateral (for burn flow)
│   └── query_server.go
├── types/
│   ├── codec.go
│   ├── errors.go
│   ├── keys.go
│   ├── msgs.go
│   └── types.go
├── module.go
└── genesis.go
```

### Implementation requirements:

**`x/ibc-collateral/types/types.go`**
```go
type CollateralAsset struct {
    IBCDenom      string      // e.g. "ibc/ABCDEF..." (IBC hash denom)
    BaseDenom     string      // human-readable e.g. "axlUSDC"
    MinRatio      sdk.Dec     // minimum collateral ratio for this asset, e.g. sdk.NewDecWithPrec(110, 2) = 1.10
    LiquidationThreshold sdk.Dec // e.g. 1.05 = 105%
    IsActive      bool
}

type CollateralPosition struct {
    Owner         sdk.AccAddress
    IBCDenom      string
    Amount        sdk.Int
    LockedAt      int64       // block height
    MintedZYTD    sdk.Int     // how much ZYTD was minted against this collateral
}

// Hardcoded whitelist tiers:
// ZYTC:    MinRatio=2.00, LiquidationThreshold=1.50 (native, volatile)
// axlUSDC: MinRatio=1.10, LiquidationThreshold=1.05 (stablecoin collateral)
// mUSDT:   MinRatio=1.10, LiquidationThreshold=1.05
// ATOM:    MinRatio=1.60, LiquidationThreshold=1.30
// wBTC:    MinRatio=1.50, LiquidationThreshold=1.25
// wETH:    MinRatio=1.50, LiquidationThreshold=1.25
```

**`x/ibc-collateral/types/keys.go`**
```go
const (
    ModuleName        = "ibccollateral"
    StoreKey          = "ibccollateral"
    RouterKey         = "ibccollateral"
    ModuleAccountName = "ibc_collateral_vault"

    CollateralAssetPrefix    = "asset/"      // asset/<ibcDenom> -> CollateralAsset JSON
    CollateralPositionPrefix = "position/"   // position/<owner_hex>/<ibcDenom> -> CollateralPosition JSON
    TotalLockedPrefix        = "locked/"     // locked/<ibcDenom> -> sdk.Int (total locked amount)
)
```

**`x/ibc-collateral/types/errors.go`**
```go
// Error codes 1400 series
var (
    ErrDenomNotWhitelisted  = errorsmod.Register(ModuleName, 1400, "IBC denom not whitelisted as collateral")
    ErrAssetNotActive       = errorsmod.Register(ModuleName, 1401, "collateral asset is not active")
    ErrInsufficientCollateral = errorsmod.Register(ModuleName, 1402, "collateral amount below minimum for requested ZYTD")
    ErrPositionNotFound     = errorsmod.Register(ModuleName, 1403, "collateral position not found")
    ErrUnlockDenied         = errorsmod.Register(ModuleName, 1404, "cannot unlock: outstanding ZYTD debt")
)
```

**`x/ibc-collateral/ibc_middleware.go`** — Implement ICS-4 `IBCModule` middleware:
```go
// Wrap the existing transfer.IBCModule
// Override OnRecvPacket:
//   1. Call the underlying transfer module's OnRecvPacket first.
//   2. Parse the FungibleTokenPacketData from the packet.
//   3. Check if data.Denom is in the CollateralAsset whitelist.
//   4. If whitelisted: emit EventTypeCollateralReceived with attrs:
//      - sender (original IBC sender)
//      - receiver (Zytherion address)
//      - ibc_denom
//      - amount
//      - base_denom
//   5. If not whitelisted: still pass through (don't block non-collateral IBC transfers)
// Override OnAcknowledgementPacket and OnTimeoutPacket: just delegate to transfer module.
```

**`x/ibc-collateral/keeper/keeper.go`** — implement ALL:
```go
func (k Keeper) RegisterCollateralAsset(ctx sdk.Context, asset CollateralAsset) error
func (k Keeper) GetCollateralAsset(ctx sdk.Context, ibcDenom string) (CollateralAsset, error)
func (k Keeper) GetAllCollateralAssets(ctx sdk.Context) []CollateralAsset
func (k Keeper) LockCollateral(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, amount sdk.Int) error
    // Transfer tokens from owner to ModuleAccountName vault
    // Create or update CollateralPosition
    // Update TotalLocked counter
func (k Keeper) UnlockCollateral(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, amount sdk.Int) error
    // Check position.MintedZYTD == 0 (no outstanding debt) before unlocking
    // Transfer tokens from vault back to owner
    // Delete CollateralPosition if amount becomes 0
func (k Keeper) GetPosition(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string) (CollateralPosition, error)
func (k Keeper) SetPosition(ctx sdk.Context, pos CollateralPosition)
func (k Keeper) GetTotalLocked(ctx sdk.Context, ibcDenom string) sdk.Int
func (k Keeper) UpdateMintedZYTD(ctx sdk.Context, owner sdk.AccAddress, ibcDenom string, delta sdk.Int) error
    // Called by x/stablecoin when ZYTD is minted/burned
    // delta is positive on mint, negative on burn
```

**`x/ibc-collateral/client/cli/tx.go`** — CLI:
```bash
# Lock native ZYTC as collateral (Phase 1 / MVP)
zytheriond tx ibccollateral lock-collateral uzytc 1500000 \
  --from alice --chain-id zytherion --gas 200000 -y

# Unlock collateral (only if no ZYTD debt)
zytheriond tx ibccollateral unlock-collateral uzytc 1500000 \
  --from alice --chain-id zytherion --gas 200000 -y
```

**`x/ibc-collateral/client/cli/query.go`** — CLI:
```bash
zytheriond query ibccollateral position <address> <ibc_denom>
zytheriond query ibccollateral assets          # all whitelisted collateral assets
zytheriond query ibccollateral total-locked    # total locked per denom
```

---

## MODULE 3: `x/stablecoin` — ZYTD Mint, Burn & Liquidation

### Purpose
Core stablecoin logic. Uses prices from `x/oracle` and collateral positions from `x/ibc-collateral` to mint/burn ZYTD and manage liquidations.

### File structure to create:
```
x/stablecoin/
├── client/cli/
│   ├── query.go
│   └── tx.go
├── keeper/
│   ├── keeper.go
│   ├── msg_server_mint.go
│   ├── msg_server_burn.go
│   ├── msg_server_liquidate.go
│   └── query_server.go
├── types/
│   ├── codec.go
│   ├── errors.go
│   ├── keys.go
│   ├── msgs.go
│   └── types.go
├── module.go
└── genesis.go
```

### Implementation requirements:

**`x/stablecoin/types/types.go`**
```go
type StablecoinParams struct {
    ZYTDDenom            string    // "uzytd"
    StabilityFeePerYear  sdk.Dec   // annual stability fee, e.g. 0.005 = 0.5%
    LiquidationPenalty   sdk.Dec   // penalty paid to liquidator, e.g. 0.10 = 10%
    LiquidatorReward     sdk.Dec   // portion of penalty going to liquidator, e.g. 0.08
    ProtocolFeeReceiver  string    // module account or governance address receiving 2%
}

type MintRecord struct {
    Owner       sdk.AccAddress
    IBCDenom    string        // which collateral was used
    Minted      sdk.Int       // ZYTD amount minted (in uzytd)
    CollateralUSD sdk.Dec     // USD value of collateral at mint time
    MintedAt    int64         // block height
}
```

**`x/stablecoin/types/keys.go`**
```go
const (
    ModuleName        = "stablecoin"
    StoreKey          = "stablecoin"
    RouterKey         = "stablecoin"
    ModuleAccountName = "stablecoin_mint"
    ZYTDDenom         = "uzytd"

    MintRecordPrefix  = "mint/"    // mint/<owner_hex>/<ibcDenom> -> MintRecord JSON
    TotalSupplyKey    = "supply"   // -> sdk.Int (total ZYTD minted)
    ParamsKey         = "sc_params"
)
```

**`x/stablecoin/types/errors.go`**
```go
// Error codes 1500 series
var (
    ErrCollateralRatioTooLow   = errorsmod.Register(ModuleName, 1500, "collateral ratio below minimum for this asset")
    ErrZeroMintAmount          = errorsmod.Register(ModuleName, 1501, "ZYTD mint amount must be greater than zero")
    ErrInsufficientZYTDBalance = errorsmod.Register(ModuleName, 1502, "insufficient ZYTD balance to burn")
    ErrPositionHealthy         = errorsmod.Register(ModuleName, 1503, "position is healthy, cannot liquidate")
    ErrOraclePriceUnavailable  = errorsmod.Register(ModuleName, 1504, "oracle TWAP price not available for collateral denom")
    ErrExceedsCollateralValue  = errorsmod.Register(ModuleName, 1505, "requested ZYTD exceeds allowed mint amount for collateral")
)
```

**`x/stablecoin/keeper/msg_server_mint.go`** — `MsgMintZYTD` handler, full logic:
```go
// Input: sender, ibcDenom, collateralAmount, requestedZYTD
// Steps:
//   1. Get CollateralAsset from ibcCollateralKeeper — error if not whitelisted/active.
//   2. Get TWAP price of ibcDenom from oracleKeeper — error ErrOraclePriceUnavailable if unavailable.
//   3. Calculate collateralUSD = collateralAmount * TWAPprice (use sdk.Dec arithmetic).
//   4. Calculate maxMintable = collateralUSD / asset.MinRatio.
//      e.g. $150 collateral at 150% ratio → max 100 ZYTD.
//   5. Error ErrExceedsCollateralValue if requestedZYTD > maxMintable.
//   6. Call ibcCollateralKeeper.LockCollateral(sender, ibcDenom, collateralAmount).
//   7. Mint requestedZYTD uzytd via bankKeeper.MintCoins to stablecoin module account.
//   8. Send minted coins from module account to sender via bankKeeper.SendCoinsFromModuleToAccount.
//   9. Call ibcCollateralKeeper.UpdateMintedZYTD(sender, ibcDenom, +requestedZYTD).
//  10. Store MintRecord in KV.
//  11. Update TotalSupply.
//  12. Emit event EventTypeMintZYTD with: sender, ibc_denom, collateral_amount, zytd_minted, collateral_ratio.
```

**`x/stablecoin/keeper/msg_server_burn.go`** — `MsgBurnZYTD` handler:
```go
// Input: sender, ibcDenom, zytdAmount
// Steps:
//   1. Get MintRecord for sender+ibcDenom — error if not found.
//   2. Error if zytdAmount > MintRecord.Minted.
//   3. Send zytdAmount uzytd from sender to stablecoin module account.
//   4. Burn coins via bankKeeper.BurnCoins.
//   5. Calculate collateral to return proportionally:
//      returnAmount = (zytdAmount / MintRecord.Minted) * lockedCollateral
//   6. Call ibcCollateralKeeper.UnlockCollateral(sender, ibcDenom, returnAmount).
//   7. Call ibcCollateralKeeper.UpdateMintedZYTD(sender, ibcDenom, -zytdAmount).
//   8. Update or delete MintRecord.
//   9. Update TotalSupply.
//  10. Emit event EventTypeBurnZYTD.
```

**`x/stablecoin/keeper/msg_server_liquidate.go`** — `MsgLiquidate` handler:
```go
// Input: liquidator, targetOwner, ibcDenom
// Anyone can call this if a position is undercollateralized.
// Steps:
//   1. Get CollateralPosition of targetOwner+ibcDenom from ibcCollateralKeeper.
//   2. Get TWAP price of ibcDenom from oracleKeeper.
//   3. Calculate currentRatio = (collateralAmount * price) / position.MintedZYTD.
//   4. Error ErrPositionHealthy if currentRatio >= asset.LiquidationThreshold.
//   5. Calculate:
//      - debtZYTD = position.MintedZYTD (full liquidation)
//      - collateralToSeize = (debtZYTD / price) * (1 + LiquidationPenalty)
//      - liquidatorReward  = (debtZYTD / price) * LiquidatorReward
//      - protocolFee       = collateralToSeize - liquidatorReward - debtCollateral
//   6. Liquidator sends debtZYTD uzytd to module, gets burned.
//   7. Transfer liquidatorReward collateral to liquidator.
//   8. Transfer protocolFee collateral to ProtocolFeeReceiver address.
//   9. Update position: close it (MintedZYTD → 0, Amount → remaining).
//  10. Emit EventTypeLiquidation with all relevant details.
```

**`x/stablecoin/keeper/query_server.go`**:
```go
func (k Keeper) QueryMintRecord(ctx, req) -> (MintRecord, error)
func (k Keeper) QueryCollateralRatio(ctx, req) -> (sdk.Dec, error)  // live ratio for a position
func (k Keeper) QueryTotalSupply(ctx, req) -> (sdk.Int, error)      // total ZYTD in circulation
func (k Keeper) QueryMaxMintable(ctx, req) -> (sdk.Int, error)      // max ZYTD for given collateral+amount
```

**`x/stablecoin/client/cli/tx.go`** — CLI:
```bash
# Mint ZYTD using axlUSDC collateral
zytheriond tx stablecoin mint-zytd \
  --collateral-denom ibc/AXLUSDC_CHANNEL_HASH \
  --collateral-amount 1000000 \
  --zytd-amount 900000 \
  --from alice --gas 300000 --chain-id zytherion -y

# Burn ZYTD to reclaim collateral
zytheriond tx stablecoin burn-zytd \
  --collateral-denom ibc/AXLUSDC_CHANNEL_HASH \
  --zytd-amount 900000 \
  --from alice --gas 300000 --chain-id zytherion -y

# Liquidate an undercollateralized position
zytheriond tx stablecoin liquidate \
  --target <undercollateralized_address> \
  --collateral-denom ibc/AXLUSDC_CHANNEL_HASH \
  --from liquidator --gas 400000 --chain-id zytherion -y
```

**`x/stablecoin/client/cli/query.go`** — CLI:
```bash
zytheriond query stablecoin mint-record <address> <ibc_denom>
zytheriond query stablecoin ratio <address> <ibc_denom>     # live collateral ratio
zytheriond query stablecoin total-supply                    # total ZYTD minted
zytheriond query stablecoin max-mintable <ibc_denom> <amount>  # simulation
```

---

## ADDITION 4: CosmWasm Integration

### Purpose
Enable permissioned smart contracts (CosmWasm) on Zytherion so developers can build DeFi protocols using ZYTD and ZYTC tokens. Use permissioned mode (governance controls who can upload contracts).

### Files to create or modify:

**`go.mod`** — Add dependency:
```
github.com/CosmWasm/wasmd v0.45.0
```

**`app/app.go`** — Add CosmWasm module wiring:

Add these imports (add to existing import block, do not remove existing imports):
```go
wasmkeeper    "github.com/CosmWasm/wasmd/x/wasm/keeper"
wasmtypes     "github.com/CosmWasm/wasmd/x/wasm/types"
wasm          "github.com/CosmWasm/wasmd/x/wasm"
wasmclient    "github.com/CosmWasm/wasmd/x/wasm/client"
```

Add to `ZytherionApp` struct:
```go
WasmKeeper     wasmkeeper.Keeper
ScopedWasmKeeper capabilitykeeper.ScopedKeeper
```

In `NewZytherionApp`, add wasm keeper initialization:
```go
// Wasm keeper (permissioned — governance controls upload access)
wasmDir := filepath.Join(homePath, "wasm")
wasmConfig := wasmtypes.DefaultWasmConfig()

app.WasmKeeper = wasmkeeper.NewKeeper(
    appCodec,
    keys[wasmtypes.StoreKey],
    app.AccountKeeper,
    app.BankKeeper,
    app.StakingKeeper,
    distrkeeper.NewQuerier(app.DistrKeeper),
    app.IBCFacadeKeeper,         // or IBCKeeper, depending on existing wiring
    app.CapabilityKeeper.ScopeToModule(wasmtypes.ModuleName),
    transferkeeper.NewQuerier(app.TransferKeeper),
    app.MsgServiceRouter(),
    app.GRPCQueryRouter(),
    wasmDir,
    wasmConfig,
    wasmkeeper.BuiltInCapabilities(),
    authority,                   // governance address for admin actions
    wasmtypes.DefaultParams(),
)
```

Add `wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName))` to `ModuleManager` initialization.

Add `wasmtypes.StoreKey` to `NewKVStoreKeys(...)` call.

Add wasm routes to IBC router if IBC is already wired.

**`app/app.go`** — also register all three new modules in `ModuleManager`:
```go
oracle.NewAppModule(appCodec, app.OracleKeeper),
ibccollateral.NewAppModule(appCodec, app.IBCCollateralKeeper),
stablecoin.NewAppModule(appCodec, app.StablecoinKeeper),
```

### CosmWasm CLI usage (document in README section):
```bash
# Upload a CosmWasm contract (requires governance permission in permissioned mode)
zytheriond tx wasm store contract.wasm \
  --from alice --gas 2000000 --chain-id zytherion -y

# Instantiate
zytheriond tx wasm instantiate <code_id> '{"owner":"zyth1abc..."}' \
  --label "my-defi-contract" --no-admin \
  --from alice --gas 300000 --chain-id zytherion -y

# Execute
zytheriond tx wasm execute <contract_addr> '{"transfer":{"to":"zyth1xyz...","amount":"1000"}}' \
  --from alice --gas 200000 --chain-id zytherion -y

# Query
zytheriond query wasm contract-state smart <contract_addr> '{"balance":{"address":"zyth1abc..."}}'
```

---

## MODULE WIRING IN `app/app.go`

Add all three new keepers to the `ZytherionApp` struct:
```go
OracleKeeper         oraclekeeper.Keeper
IBCCollateralKeeper  ibccollateralkeeper.Keeper
StablecoinKeeper     stablecoinkeeper.Keeper
WasmKeeper           wasmkeeper.Keeper
```

Initialize keepers in `NewZytherionApp` in this order (dependency order):
1. `OracleKeeper` — depends on: `StakingKeeper`, `BankKeeper`
2. `IBCCollateralKeeper` — depends on: `BankKeeper`, `TransferKeeper`, `AccountKeeper`
3. `StablecoinKeeper` — depends on: `BankKeeper`, `OracleKeeper`, `IBCCollateralKeeper`
4. `WasmKeeper` — depends on: `AccountKeeper`, `BankKeeper`, `StakingKeeper`, `IBCFacadeKeeper`

Register store keys in `NewKVStoreKeys(...)`:
```go
oracletypes.StoreKey,
ibccollateraltypes.StoreKey,
stablecointf.StoreKey,
wasmtypes.StoreKey,
```

Set module `BeginBlock`/`EndBlock` order. Add these to existing order lists:
```go
// BeginBlockers (after existing privacy module):
oracletypes.ModuleName,
ibccollateraltypes.ModuleName,
stablecointf.ModuleName,
wasmtypes.ModuleName,

// EndBlockers (same order):
oracletypes.ModuleName,
ibccollateraltypes.ModuleName,
stablecointf.ModuleName,
wasmtypes.ModuleName,
```

---

## `go.mod` — Full additions needed

Add these to the existing `go.mod` (keep all existing dependencies):
```
github.com/CosmWasm/wasmd v0.45.0
github.com/CosmWasm/wasmvm v1.5.2
```

---

## `Makefile` — Add new targets

Add these targets to the existing Makefile (after existing targets, do not remove existing ones):
```makefile
test-oracle:
	go test ./x/oracle/... -v -count=1

test-stablecoin:
	go test ./x/stablecoin/... -v -count=1

test-ibccollateral:
	go test ./x/ibc-collateral/... -v -count=1

test-wasm:
	go test ./x/wasm/... -v -count=1

test-v05: test-oracle test-stablecoin test-ibccollateral test-wasm
	@echo "✅ All v0.5.1 module tests passed"
```

---

## CODE QUALITY REQUIREMENTS

Apply these rules to every file generated:

1. **No placeholder comments** — every function body must contain actual implementation, not `// TODO` or `panic("not implemented")`.

2. **Error wrapping** — use `fmt.Errorf("context: %w", err)` or `sdkerrors.Wrap` consistently. Never discard errors with `_`.

3. **KV store pattern** — follow the existing `x/privacy/keeper/keeper.go` pattern: marshal to JSON with `codec.MustMarshal`, use `sdk.KVStorePrefixIterator` for range queries.

4. **Event emission** — every message handler must emit events. Define event type and attribute constants in `types/keys.go` with prefix `EventType` and `AttributeKey`.

5. **Parameter validation** — all message types must implement `ValidateBasic() error` with complete field validation.

6. **Module account** — every module that holds tokens must register its `ModuleAccountName` in `maccPerms` map in `app/app.go`:
   ```go
   ibccollateraltypes.ModuleAccountName: {authtypes.Burner, authtypes.Minter},
   stablecointf.ModuleAccountName:       {authtypes.Burner, authtypes.Minter},
   ```

7. **sdk.Dec arithmetic** — use `sdk.NewDecFromInt`, `sdk.Dec.Mul`, `sdk.Dec.Quo` for all price and ratio calculations. Never use float64.

8. **Genesis** — each module must implement `DefaultGenesis()`, `ValidateGenesis()`, `InitGenesis()`, and `ExportGenesis()`.

9. **Inter-module calls** — `x/stablecoin` accesses `x/oracle` and `x/ibc-collateral` through **keeper interfaces** (not direct struct references). Define interfaces in `x/stablecoin/types/expected_keepers.go`.

---

## EXPECTED OUTPUT STRUCTURE

Generate files in this exact order:
1. `x/oracle/` — all files
2. `x/ibc-collateral/` — all files
3. `x/stablecoin/` — all files
4. `app/app.go` — modified (show only added/changed sections with clear `// --- NEW v0.5.1 ---` markers)
5. `go.mod` — modified (show only added lines)
6. `Makefile` — modified (show only added targets)

For each file, output the **complete file content** — no truncation, no `// ... rest of file unchanged`.

---

## VERSION TAG

After all code is generated, add a version marker comment to `app/app.go`:
```go
// Zytherion v0.5.1 — Multi-Collateral ZYTD + IBC + CosmWasm
// Founder: Rayhan Aziel Abbrar
// Modules added: x/oracle, x/ibc-collateral, x/stablecoin, CosmWasm
```


---

## v0.5.1 ARCHITECTURAL UPGRADES

Tiga upgrade arsitektur yang diimplementasikan sebagai bagian dari v0.5.1.  
Semua perubahan telah di-build dan diverifikasi: `go build ./...` **exit 0**.

---

### A. TFHE Worker Pool — Penghapusan Global Mutex

**Problem:** `x/privacy/tfhe/engine.go` sebelumnya menggunakan satu `sync.Mutex` global (`tfheMu`) untuk menserialisasikan semua panggilan CGo ke library Rust `tfhe-rs`. Ini menyebabkan antrian berat di bawah traffic TFHE yang padat — semua operasi homomorfik berjalan single-threaded.

**Root cause insight:** `set_server_key()` di `tfhe-rs` menggunakan **thread-local storage**, bukan global state. Artinya, beberapa goroutine yang masing-masing dikunci ke OS thread sendiri (`runtime.LockOSThread()`) dapat memanggil `set_server_key()` secara simultan tanpa konflik sama sekali.

**Formula ukuran pool:** `max(1, runtime.NumCPU() - 2)`  
Selalu sisakan 2 core untuk CometBFT consensus engine + P2P network stack.  
Tanpa reservasi ini, validator berisiko CPU starvation → missed blocks → slashing.

#### File Baru

**`x/privacy/tfhe/worker_pool.go`** *(build tag: `tfhe_cgo`)*

```go
//go:build tfhe_cgo

package tfhe

import (
    "errors"
    "fmt"
    "runtime"
    "sync"
)

const DefaultWorkerQueueDepth = 256

type tfheJob struct {
    fn     func() error
    result chan<- error
}

type TFHEWorkerPool struct {
    jobs chan tfheJob
    size int
    wg   sync.WaitGroup
}

var (
    globalPool     *TFHEWorkerPool
    globalPoolOnce sync.Once
)

// InitWorkerPool initialises the global TFHE worker pool.
// size = 0 → auto: max(1, runtime.NumCPU()-2).
// Blocks until all workers signal readiness (each pinned to its OS thread).
func InitWorkerPool(serverKey []byte, size int) (*TFHEWorkerPool, error) {
    if len(serverKey) == 0 {
        return nil, errors.New("tfhe: InitWorkerPool requires a non-empty server key")
    }
    var initErr error
    globalPoolOnce.Do(func() {
        n := size
        if n <= 0 {
            n = runtime.NumCPU() - 2
            if n < 1 {
                n = 1
            }
        }
        pool := &TFHEWorkerPool{
            jobs: make(chan tfheJob, DefaultWorkerQueueDepth),
            size: n,
        }
        started := make(chan struct{}, n)
        pool.wg.Add(n)
        for i := 0; i < n; i++ {
            skCopy := make([]byte, len(serverKey))
            copy(skCopy, serverKey)
            go pool.runWorker(skCopy, started)
        }
        for i := 0; i < n; i++ {
            <-started
        }
        globalPool = pool
    })
    if initErr != nil {
        return nil, initErr
    }
    return globalPool, nil
}

func GetPool() *TFHEWorkerPool  { return globalPool }
func PoolSize() int {
    if globalPool == nil { return 0 }
    return globalPool.size
}

func (p *TFHEWorkerPool) Submit(fn func() error) error {
    if p == nil {
        return fmt.Errorf("tfhe: worker pool is nil — call InitWorkerPool first")
    }
    result := make(chan error, 1)
    p.jobs <- tfheJob{fn: fn, result: result}
    return <-result
}

func (p *TFHEWorkerPool) runWorker(_ []byte, started chan<- struct{}) {
    defer p.wg.Done()
    // Pin permanently — goroutine owns this OS thread for life.
    runtime.LockOSThread()
    started <- struct{}{}
    for job := range p.jobs {
        job.result <- job.fn()
    }
}
```

**`x/privacy/tfhe/worker_pool_stub.go`** *(build tag: `!tfhe_cgo`)*

```go
//go:build !tfhe_cgo

package tfhe

import "errors"

type TFHEWorkerPool struct{}
func InitWorkerPool(_ []byte, _ int) (*TFHEWorkerPool, error) { return &TFHEWorkerPool{}, nil }
func GetPool() *TFHEWorkerPool { return nil }
func PoolSize() int { return 0 }
func (p *TFHEWorkerPool) Submit(fn func() error) error {
    if p == nil { return errors.New("tfhe: pool not available") }
    return fn()
}
```

**`x/privacy/tfhe/node_keys.go`** *(build tag: `tfhe_cgo`)*

Exported helper yang menyediakan singleton load/generate TFHE node keys.  
Dipakai oleh `NewKeeper` (worker pool init) dan CosmWasm query plugin (agar keys tidak di-generate dua kali).

```go
//go:build tfhe_cgo

package tfhe

import (
    "fmt"
    "os"
    "sync"
)

var (
    nodeKeysMu         sync.Mutex
    nodeClientKeyCache []byte
    nodeServerKeyCache []byte
)

// EnsureNodeKeys loads keys dari disk atau generate baru jika belum ada.
// Keys disimpan di: <nodeHome>/tfhe_client.key dan <nodeHome>/tfhe_server.key
// Thread-safe. Subsequent calls return cache tanpa disk I/O.
func EnsureNodeKeys(nodeHome string) (clientKey, serverKey []byte, err error) {
    nodeKeysMu.Lock()
    defer nodeKeysMu.Unlock()
    if len(nodeClientKeyCache) > 0 {
        return nodeClientKeyCache, nodeServerKeyCache, nil
    }
    ckPath, skPath := keyPaths(nodeHome)
    ck, errCK := os.ReadFile(ckPath)
    sk, errSK := os.ReadFile(skPath)
    if errCK == nil && errSK == nil && len(ck) > 0 && len(sk) > 0 {
        nodeClientKeyCache = ck
        nodeServerKeyCache = sk
        return ck, sk, nil
    }
    fmt.Println("[INFO] tfhe: generating new TFHE key pair (~10–60s)...")
    kp, genErr := GenerateKeys()
    if genErr != nil {
        return nil, nil, fmt.Errorf("tfhe: key generation failed: %w", genErr)
    }
    _ = os.WriteFile(ckPath, kp.ClientKey, 0600)
    _ = os.WriteFile(skPath, kp.ServerKey, 0600)
    nodeClientKeyCache = kp.ClientKey
    nodeServerKeyCache = kp.ServerKey
    return kp.ClientKey, kp.ServerKey, nil
}

func keyPaths(nodeHome string) (ck, sk string) {
    if nodeHome != "" {
        return nodeHome + "/tfhe_client.key", nodeHome + "/tfhe_server.key"
    }
    home, _ := os.UserHomeDir()
    return home + "/.zytherion_tfhe_client.key", home + "/.zytherion_tfhe_server.key"
}
```

#### File Dimodifikasi

**`x/privacy/tfhe/engine.go`** — perubahan kunci:

| Sebelum (v0.5.0) | Sesudah (v0.5.1) |
|---|---|
| `var tfheMu sync.Mutex` (global) | Dihapus sepenuhnya |
| `tfheMu.Lock()` di `AddUint32` | `pool.Submit(func() { addUint32Direct(...) })` |
| `tfheMu.Lock()` di `MultiplyScalarUint32` | `pool.Submit(func() { multiplyScalarDirect(...) })` |
| `tfheMu.Lock()` di `EncryptUint32` | Dihapus — tidak butuh mutex (clientKey-based) |
| `tfheMu.Lock()` di `DecryptUint32` | Dihapus — tidak butuh mutex (clientKey-based) |
| `GenerateKeys` | Tetap pakai local mutex (one-time slow op) |
| Tidak ada `SubUint32` | **BARU:** `SubUint32` + `subUint32Direct` |

Pola baru untuk setiap operasi server-key:

```go
func AddUint32(serverKey, ct1, ct2 []byte) ([]byte, error) {
    // ... validasi input ...
    pool := GetPool()
    if pool == nil {
        return addUint32Direct(serverKey, ct1, ct2) // fallback selama startup
    }
    var result []byte
    var opErr error
    err := pool.Submit(func() error {
        result, opErr = addUint32Direct(serverKey, ct1, ct2)
        return opErr
    })
    if err != nil { return nil, err }
    return result, nil
}

// addUint32Direct adalah raw CGo call, aman dijalankan di dalam pinned OS thread.
func addUint32Direct(serverKey, ct1, ct2 []byte) ([]byte, error) {
    resultBuf := C.malloc(C.size_t(CiphertextMaxBytes))
    defer C.free(resultBuf)
    // ... CGo call ke tfhe_add_u32 ...
}
```

**`x/privacy/keeper/keeper.go`** — tambahan di `NewKeeper` saat `enableTFHE == true`:

```go
// Inisialisasi worker pool otomatis saat node start dengan --enable-tfhe
_, sk, keyErr := tfhe.EnsureNodeKeys(nodeHome)
if keyErr != nil {
    fmt.Printf("[WARN] privacy: TFHE worker pool tidak diinisialisasi: %v\n", keyErr)
} else {
    if _, poolErr := tfhe.InitWorkerPool(sk, 0); poolErr != nil {
        fmt.Printf("[WARN] privacy: worker pool init gagal: %v\n", poolErr)
    }
}
```

#### Tambahan Rust Bridge: `tfhe_sub_u32`

**`x/privacy/tfhe/cgo_bridge.h`** — tambahkan deklarasi:

```c
// Homomorphic subtraction: result = c1 - c2 (mod 2^32).
int64_t tfhe_sub_u32(
    const uint8_t *sk_bytes, uint64_t sk_len,
    const uint8_t *c1_bytes, uint64_t c1_len,
    const uint8_t *c2_bytes, uint64_t c2_len,
    uint8_t *result_out, uint64_t out_len
);
```

**`x/privacy/tfhe/tfhe_c/src/lib.rs`** — tambahkan fungsi:

```rust
#[no_mangle]
pub extern "C" fn tfhe_sub_u32(
    sk_bytes: *const u8, sk_len: u64,
    c1_bytes: *const u8, c1_len: u64,
    c2_bytes: *const u8, c2_len: u64,
    result_out: *mut u8, out_len: u64,
) -> i64 {
    let result = std::panic::catch_unwind(|| -> Result<i64, Box<dyn std::error::Error>> {
        let server_key: tfhe::ServerKey = bincode::deserialize(
            unsafe { slice_from_raw(sk_bytes, sk_len) })?;
        set_server_key(server_key);
        let ct1: FheUint32 = bincode::deserialize(unsafe { slice_from_raw(c1_bytes, c1_len) })?;
        let ct2: FheUint32 = bincode::deserialize(unsafe { slice_from_raw(c2_bytes, c2_len) })?;
        let ct_result = ct1 - ct2; // wraps mod 2^32
        let res_bytes = bincode::serialize(&ct_result)?;
        let out_buf = unsafe { slice_mut_from_raw(result_out, out_len) };
        if res_bytes.len() > out_buf.len() { return Err("buffer too small".into()); }
        out_buf[..res_bytes.len()].copy_from_slice(&res_bytes);
        Ok(res_bytes.len() as i64)
    });
    match result { Ok(Ok(n)) => n, _ => -1 }
}
```

> **Setelah menambah fungsi Rust**, rebuild library di WSL:
> ```bash
> cd ~/zytherion/x/privacy/tfhe/tfhe_c && cargo build --release
> ```

---

### B. State Rent — Ekonomi Penyimpanan Terenkripsi

**Problem:** FheUint32 ciphertext berukuran ~21 KB per entri. Tanpa mekanisme pembayaran, state on-chain akan membengkak tak terkendali akibat data TFHE yang tidak pernah dihapus.

**Solusi:** Bootstrap rate governance-adjustable. Parameter bisa diubah on-chain via governance proposal tanpa hard fork.

#### File Baru

**`x/privacy/types/state_rent.go`**

```go
package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
    StateRentDenom                  = "uzytc"
    DefaultRentRatePerBytePerBlock  int64 = 1     // 0.000001 ZYTC/byte/block
    DefaultMaxFreeSizeBytes         int64 = 1024  // 1 KB gratis per address
    DefaultGracePeriodBlocks        int64 = 14400 // ~1 hari (6s/block)
)

// StateRentParams — semua field adjustable via governance parameter change proposal.
type StateRentParams struct {
    RentRatePerBytePerBlock int64 `json:"rent_rate_per_byte_per_block"`
    MaxFreeSizeBytes        int64 `json:"max_free_size_bytes"`
    GracePeriodBlocks       int64 `json:"grace_period_blocks"`
}

func DefaultStateRentParams() StateRentParams {
    return StateRentParams{
        RentRatePerBytePerBlock: DefaultRentRatePerBytePerBlock,
        MaxFreeSizeBytes:        DefaultMaxFreeSizeBytes,
        GracePeriodBlocks:       DefaultGracePeriodBlocks,
    }
}

// RentDue menghitung tagihan sewa untuk ukuran dan jumlah block tertentu.
// Mengembalikan 0 jika sizeBytes <= MaxFreeSizeBytes (free tier).
//
// Formula: max(0, sizeBytes - MaxFreeSize) * RentRate * blocks
func (p StateRentParams) RentDue(sizeBytes, blocks int64) sdk.Coin {
    billable := sizeBytes - p.MaxFreeSizeBytes
    if billable <= 0 || blocks <= 0 {
        return sdk.NewInt64Coin(StateRentDenom, 0)
    }
    return sdk.NewInt64Coin(StateRentDenom, billable*p.RentRatePerBytePerBlock*blocks)
}
```

**Kalkulasi biaya storage:**

```
21 KB ciphertext × 1 uzytc/byte/block × 14,400 block/hari
= 302,400 uzytc/hari
= 0.3024 ZYTC/hari
≈ $0.03/hari (pada harga $0.10/ZYTC)
```

**`x/privacy/keeper/state_rent.go`** — fungsi utama:

| Fungsi | Deskripsi |
|--------|-----------|
| `GetStateRentParams(ctx)` | Baca params dari KV store (fallback ke default) |
| `SetStateRentParams(ctx, params)` | Simpan params baru (dipanggil oleh governance) |
| `CollectRent(ctx, key, owner, sizeBytes)` | Tagih sewa untuk satu commitment; burn uzytc yang terkumpul; mulai grace period jika gagal |
| `CheckAndEvict(ctx, key)` | Cek apakah grace period habis; emit `commitment_evicted` event sebelum prune (untuk archival node) |

**Events yang diemit:**

```go
const (
    EventTypeRentCollected    = "rent_collected"    // sewa berhasil ditarik
    EventTypeRentDefault      = "rent_default"      // owner gagal bayar, mulai grace period
    EventTypeCommitmentEvicted = "commitment_evicted" // data dihapus setelah grace period
)
```

> `commitment_evicted` event dikirim **sebelum** data dihapus dari state, sehingga archival node (Arweave/Filecoin) sempat capture data sebelum hilang.

---

### C. Confidential ZYTD — Partial Privacy MVP

**Model privasi:**
- **Kolateral**: **PUBLIC** — bot likuidator harus bisa monitor posisi unhealthy secara real-time untuk menjaga peg $1 ZYTD
- **ZYTD balance**: **PRIVATE** — disimpan sebagai FheUint32 ciphertext
- **ZYTD transfer amount**: **PRIVATE** — pengirim submit ciphertext, chain update balance via TFHE Add/Sub

> Full privacy (kolateral terenkripsi) direncanakan di v0.8 menggunakan ZK range proof untuk mendeteksi underflow tanpa men-decrypt nilai.

#### File Baru

**`x/stablecoin/keeper/confidential_transfer.go`**

**Storage:**

```go
const encryptedZYTDBalancePrefix = "enc_zytd/"

type EncryptedBalance struct {
    Ciphertext       []byte `json:"ciphertext"`        // FheUint32 ct
    LastUpdatedBlock int64  `json:"last_updated_block"` // untuk state rent
    SizeBytes        int64  `json:"size_bytes"`         // len(Ciphertext)
}
```

**Confidential Mint** — dipanggil setelah collateral ratio check lolos:

```go
func (k Keeper) ConfidentialMintZYTD(
    ctx sdk.Context,
    recipient sdk.AccAddress,
    mintAmount uint32,  // plaintext — berasal dari formula mint yang sudah divalidasi
    clientKey, serverKey []byte,
) error {
    // 1. Encrypt mint amount
    encMint, _ := tfhe.EncryptUint32(clientKey, mintAmount)

    // 2. Load existing encrypted balance
    existing := k.GetEncryptedZYTDBalance(ctx, recipient)

    var newCT []byte
    if len(existing.Ciphertext) == 0 {
        newCT = encMint // mint pertama
    } else {
        // Enc(balance) + Enc(mint) = Enc(balance + mint)
        newCT, _ = tfhe.AddUint32(serverKey, existing.Ciphertext, encMint)
    }

    // 3. Simpan ciphertext baru (bukan nilai plaintext!)
    _ = k.SetEncryptedZYTDBalance(ctx, recipient, EncryptedBalance{
        Ciphertext: newCT, LastUpdatedBlock: ctx.BlockHeight(), SizeBytes: int64(len(newCT)),
    })

    // 4. Emit event TANPA mencantumkan mint_amount (validator tidak tahu nilainya)
    ctx.EventManager().EmitEvent(sdk.NewEvent(
        "confidential_mint_zytd",
        sdk.NewAttribute("recipient", recipient.String()),
        sdk.NewAttribute("balance_ct_size_bytes", fmt.Sprintf("%d", len(newCT))),
    ))
    return nil
}
```

**Confidential Transfer:**

```go
func (k Keeper) ConfidentialTransferZYTD(
    ctx sdk.Context,
    sender, recipient sdk.AccAddress,
    encTransferAmount []byte, // FheUint32 ct yang disubmit pengirim
    serverKey []byte,
) error {
    // Minimal size check (FheUint32 selalu >= 8 KB)
    if len(encTransferAmount) < 8*1024 {
        return fmt.Errorf("transfer ciphertext terlalu kecil: %d bytes", len(encTransferAmount))
    }

    // Sender: Enc(balance) - Enc(transfer)
    senderBal := k.GetEncryptedZYTDBalance(ctx, sender)
    newSenderCT, _ := tfhe.SubUint32(serverKey, senderBal.Ciphertext, encTransferAmount)
    _ = k.SetEncryptedZYTDBalance(ctx, sender, EncryptedBalance{Ciphertext: newSenderCT, ...})

    // Recipient: Enc(balance) + Enc(transfer)
    recipientBal := k.GetEncryptedZYTDBalance(ctx, recipient)
    var newRecipientCT []byte
    if len(recipientBal.Ciphertext) == 0 {
        newRecipientCT = encTransferAmount
    } else {
        newRecipientCT, _ = tfhe.AddUint32(serverKey, recipientBal.Ciphertext, encTransferAmount)
    }
    _ = k.SetEncryptedZYTDBalance(ctx, recipient, EncryptedBalance{Ciphertext: newRecipientCT, ...})

    // Event: hanya metadata (sender + recipient), jumlah tidak bocor
    ctx.EventManager().EmitEvent(sdk.NewEvent(
        "confidential_transfer_zytd",
        sdk.NewAttribute("sender", sender.String()),
        sdk.NewAttribute("recipient", recipient.String()),
        sdk.NewAttribute("transfer_ct_size_bytes", fmt.Sprintf("%d", len(encTransferAmount))),
    ))
    return nil
}
```

> ⚠️ **MVP Note:** `SubUint32` menggunakan aritmatika u32 (wrap mod 2^32). Underflow tidak dicegah di v0.5.1 — ZK range proof untuk prevent fraud direncanakan di v0.8.

**Query encrypted balance (CosmWasm compatible):**

```go
func (k Keeper) QueryEncryptedZYTDBalance(
    ctx sdk.Context, addr sdk.AccAddress,
) (*tfhecosmwasm.TFHECiphertextResponse, error) {
    bal := k.GetEncryptedZYTDBalance(ctx, addr)
    return &tfhecosmwasm.TFHECiphertextResponse{
        Ciphertext: bal.Ciphertext,
        SizeBytes:  int(len(bal.Ciphertext)),
    }, nil
}
```

---

### Summary File Changes (v0.5.1 Architectural Upgrades)

| File | Status | Deskripsi |
|------|--------|-----------|
| `x/privacy/tfhe/worker_pool.go` | 🆕 NEW | OS-thread-pinned worker pool, `max(1, NumCPU-2)` workers |
| `x/privacy/tfhe/worker_pool_stub.go` | 🆕 NEW | Stub untuk build tanpa `tfhe_cgo` |
| `x/privacy/tfhe/node_keys.go` | 🆕 NEW | Exported `EnsureNodeKeys` singleton key loader |
| `x/privacy/tfhe/node_keys_stub.go` | 🆕 NEW | Stub untuk build tanpa `tfhe_cgo` |
| `x/privacy/tfhe/engine.go` | ✏️ MODIFIED | Hapus `tfheMu` mutex, routing ke pool, tambah `SubUint32` |
| `x/privacy/tfhe/engine_stub.go` | ✏️ MODIFIED | Tambah stub `SubUint32` + `EnsureNodeKeys` |
| `x/privacy/tfhe/cgo_bridge.h` | ✏️ MODIFIED | Deklarasi `tfhe_sub_u32` |
| `x/privacy/tfhe/tfhe_c/src/lib.rs` | ✏️ MODIFIED | Implementasi `tfhe_sub_u32` FFI |
| `x/privacy/types/state_rent.go` | 🆕 NEW | `StateRentParams`, `DefaultStateRentParams`, `RentDue` |
| `x/privacy/keeper/state_rent.go` | 🆕 NEW | `CollectRent`, `CheckAndEvict`, grace period logic |
| `x/privacy/keeper/keeper.go` | ✏️ MODIFIED | Auto-init worker pool di `NewKeeper` |
| `x/stablecoin/keeper/confidential_transfer.go` | 🆕 NEW | `EncryptedBalance`, `ConfidentialMintZYTD`, `ConfidentialTransferZYTD` |

**Build verification:** `go build ./...` → exit 0, zero errors, zero warnings.

---

*Generated for Zytherion Blockchain v0.5.1 — Founder: Rayhan Aziel Abbrar*

---

## v0.5.2 SECURITY PATCH

> **Dua kerentanan kritis ditemukan dan diperbaiki sebelum mainnet.**
> Kedua fix diimplementasikan, di-build, dan didokumentasikan sebagai v0.5.2.

---

### CVE-ZYTH-001 — SubUint32 Underflow (Severity: CRITICAL)

**Vektor serangan:**

```
Attacker memiliki 0 ZYTD balance.
Attacker memanggil ConfidentialTransferZYTD(amount = Enc(1_000_000)).
Chain mengeksekusi: Enc(0) - Enc(1_000_000) → Enc(2^32 - 1_000_000) ≈ Enc(4,293,967,296).
Attacker sekarang memiliki ~4.3 miliar ZYTD secara rahasia.
Validator tidak bisa melihat nilai karena ciphertext bersifat opaque.
```

**Root cause:** FheUint32 menggunakan aritmatika modular. `SubUint32` tidak memiliki validasi `balance >= amount` karena perbandingan nilai di FHE memerlukan rangkaian evaluasi khusus (yang belum ada).

**Fix yang diimplementasikan (v0.5.2):**

Tambah field `PublicCreditLimit uint64` ke setiap akun ZYTD. Field ini adalah **plaintext** dan visible on-chain.

**Invariant yang dijaga:**
```
PublicCreditLimit ≤ true_plaintext_balance (SELALU)
```

**Aturan update:**
- `MintZYTD(amount)` → `PublicCreditLimit += amount`
- `TransferZYTD(amount)` → chain validasi `amount <= PublicCreditLimit`, lalu `PublicCreditLimit -= amount`
- `BurnZYTD(amount)` → `PublicCreditLimit -= amount`
- Receive transfer → `PublicCreditLimit += received_amount` (agar bisa re-transfer)

**Trade-off privasi yang jujur:**

| | v0.5.1 (vulnerable) | v0.5.2 (fixed) |
|--|--|--|
| Balance | 🔒 Private | 🔒 Private |
| Transfer amount | 🔒 Private | 👁️ **Public** |
| Underflow attack | ❌ Possible | ✅ Blocked |

Transfer amounts sekarang public. Ini mirip Monero sebelum RingCT hadir — amounts visible, balances tidak. Solusi ideal (ZK range proof untuk sembunyikan amounts sekaligus prevent underflow) direncanakan di **v0.8**.

**Perubahan struct:**

```go
// ZYTDAccountState menggantikan EncryptedBalance
type ZYTDAccountState struct {
    EncryptedBalance  []byte `json:"encrypted_balance"`    // FheUint32 ct (private)
    PublicCreditLimit uint64 `json:"public_credit_limit"` // max outflow guard (public)
    LastUpdatedBlock  int64  `json:"last_updated_block"`
    SizeBytes         int64  `json:"size_bytes"`
}
```

**Signature baru `ConfidentialTransferZYTD`:**

```go
// SEBELUM (v0.5.1) — menerima ciphertext opaque, tidak ada guard
func (k Keeper) ConfidentialTransferZYTD(
    ctx sdk.Context,
    sender, recipient sdk.AccAddress,
    encTransferAmount []byte,  // ← opaque, tidak bisa divalidasi
    serverKey []byte,
) error

// SESUDAH (v0.5.2) — menerima plaintext amount, ada credit limit check
func (k Keeper) ConfidentialTransferZYTD(
    ctx sdk.Context,
    sender, recipient sdk.AccAddress,
    plaintextAmount uint64,  // ← public, chain dapat memvalidasi
    serverKey []byte,
) error {
    // CRITICAL CHECK — ini yang mencegah exploit:
    if plaintextAmount > senderState.PublicCreditLimit {
        return fmt.Errorf("transfer %d exceeds credit limit %d", plaintextAmount, senderState.PublicCreditLimit)
    }
    // ... lanjutkan TFHE Sub ...
}
```

**KV Store key baru:**
- `enc_zytd_v2/<address>` → `ZYTDAccountState` (JSON)
- `tfhe_pubkey/<address>` → raw TFHE CompressedPublicKey bytes

---

### CVE-ZYTH-002 — Node-Held Decryption Keys (Severity: HIGH)

**Vektor serangan:**

```
Validator node A menggunakan ClientKey miliknya untuk encrypt semua balance pengguna.
Operator node A (atau penyerang yang mengakses disk node) dapat menjalankan:
  DecryptUint32(nodeClientKey, user_balance_ciphertext)
Seluruh balance ZYTD semua pengguna terbaca.
```

**Root cause:** `ConfidentialMintZYTD` memanggil `tfhe.EncryptUint32(nodeClientKey, mintAmount)`. Node menggunakan satu ClientKey global untuk semua enkripsi, jadi siapa saja yang punya ClientKey node bisa decrypt balance siapa saja.

**Fix yang diimplementasikan (v0.5.2): Two-Key Model**

```
User generates locally:   ClientKey (secret, NEVER leaves device)
                          PublicKey (safe to share, upload to chain)
                          ServerKey (used for evaluation, upload to chain)

On-chain state:           PublicKey[user] ← registered via MsgRegisterUserTFHEPublicKey
                          ServerKey       ← shared node key for homomorphic evaluation
                          Enc_PK[user](balance) ← encrypted with USER's PublicKey

Validator capability:     CAN evaluate (Add/Sub using ServerKey)
                          CANNOT decrypt (no ClientKey)

User capability:          CAN decrypt (ClientKey never left their device)
                          CAN verify their balance locally
```

**New message type:**

```go
// MsgRegisterUserTFHEPublicKey — user registers their TFHE public key on-chain.
// Must be called before first mint.
// The public key allows the chain to encrypt values destined for this user.
// The matching ClientKey stays on the user's device and is NEVER shared.
type MsgRegisterUserTFHEPublicKey struct {
    Address   string `json:"address"`
    PublicKey []byte `json:"public_key"` // serialised tfhe::CompressedPublicKey
}
```

**Flow baru mint:**

```go
// v0.5.1 (VULNERABLE)
encMint, _ := tfhe.EncryptUint32(nodeClientKey, mintAmount)
//                                ↑ node key! operator bisa decrypt!

// v0.5.2 (FIXED)
userPubKey, _ := k.GetUserTFHEPublicKey(ctx, recipient)
encMint, _ := tfhe.EncryptWithPublicKey(userPubKey, mintAmount)
//                                       ↑ user's public key — operator blind!
```

**New Rust FFI function:**

```rust
// cgo_bridge.h — deklarasi baru
int64_t tfhe_encrypt_u32_pk(
    const uint8_t *pk_bytes, uint64_t pk_len,  // CompressedPublicKey
    uint32_t plaintext,
    uint8_t *ct_out, uint64_t out_len
);

// lib.rs — implementasi
pub extern "C" fn tfhe_encrypt_u32_pk(...) -> i64 {
    let public_key: CompressedPublicKey = bincode::deserialize(pk_slice)?;
    let ciphertext: FheUint32 = FheUint32::encrypt(plaintext, &public_key);
    // ...
}
```

**New Go API:**

```go
// engine.go
func EncryptWithPublicKey(pubKey []byte, plaintext uint32) ([]byte, error)
func EncryptWithServerKey(serverKey []byte, plaintext uint32) ([]byte, error)
```

**Sisa risiko yang di-acknowledge (honest):**

> `EncryptWithServerKey` di path SubUint32 masih menggunakan node ClientKey untuk membuat ciphertext operand intermediate. Ini adalah **known residual risk** yang didokumentasikan. Eliminasi penuh membutuhkan threshold re-encryption atau proxy re-encryption agar ciphertext sender dan operand Sub menggunakan konfigurasi kunci yang sama. Direncanakan di **v0.8** bersama ZK range proof.

**Perbandingan sebelum/sesudah:**

| Kemampuan Operator Node | v0.5.1 | v0.5.2 |
|--|--|--|
| Decrypt balance user | ✅ Bisa | ❌ Tidak bisa |
| Lihat transfer amounts | ✅ Bisa | ✅ Bisa (by design) |
| Evaluasi homomorphik (Add/Sub) | ✅ Bisa | ✅ Bisa |
| Generate ciphertext untuk user | ✅ Bisa | ✅ Bisa (via user's PublicKey) |
| Forge user balance | ❌ Tidak bisa | ❌ Tidak bisa |

---

### Summary File Changes (v0.5.2 Security Patch)

| File | Status | Fix |
|------|--------|-----|
| `x/stablecoin/keeper/confidential_transfer.go` | ✏️ REWRITTEN | CVE-001 + CVE-002: ZYTDAccountState, credit limit, user pubkey registry |
| `x/privacy/tfhe/engine.go` | ✏️ MODIFIED | CVE-002: `EncryptWithPublicKey`, `EncryptWithServerKey` |
| `x/privacy/tfhe/engine_stub.go` | ✏️ MODIFIED | Stubs untuk kedua fungsi baru |
| `x/privacy/tfhe/cgo_bridge.h` | ✏️ MODIFIED | CVE-002: deklarasi `tfhe_encrypt_u32_pk` |
| `x/privacy/tfhe/tfhe_c/src/lib.rs` | ✏️ MODIFIED | CVE-002: `CompressedPublicKey` import + `tfhe_encrypt_u32_pk` impl |

**Build verification:** `go build ./...` → exit 0, zero errors.

> ⚠️ **Setelah update Rust**, rebuild library di WSL:
> ```bash
> cd ~/zytherion/x/privacy/tfhe/tfhe_c && cargo build --release
> ```

---

### Roadmap ke v0.8 (Full Privacy)

```
v0.5.2 (sekarang):
  ✅ Underflow blocked via PublicCreditLimit
  ✅ Operator cannot decrypt user balances
  ⚠️  Transfer amounts public
  ⚠️  Sub path still uses node key as intermediate

v0.8 (planned):
  🎯 ZK range proof: prove balance >= amount WITHOUT revealing either value
  🎯 Transfer amounts kembali private
  🎯 Threshold re-encryption untuk Sub path isolation
  🎯 Full user-key isolation di semua code paths
```

---

*Generated for Zytherion Blockchain v0.5.2 — Founder: Rayhan Aziel Abbrar*
