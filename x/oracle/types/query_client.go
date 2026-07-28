package types

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"google.golang.org/grpc"
)

// QueryClient is a thin gRPC-agnostic query client for the oracle module.
// Since we use amino routing rather than full protobuf gRPC, this client
// communicates via the Cosmos SDK's ABCI query mechanism.
type QueryClient struct {
	clientCtx client.Context
}

// NewQueryClient creates a new oracle QueryClient from a Cosmos client context.
func NewQueryClient(clientCtx client.Context) *QueryClient {
	return &QueryClient{clientCtx: clientCtx}
}

// QueryPrice queries the latest oracle price for a denom via ABCI.
func (c *QueryClient) QueryPrice(_ context.Context, req *QueryPriceRequest, _ ...grpc.CallOption) (*QueryPriceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	res, _, err := c.clientCtx.QueryWithData(
		fmt.Sprintf("custom/%s/price/%s", ModuleName, req.Denom),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("query price: %w", err)
	}
	var resp QueryPriceResponse
	if err := json.Unmarshal(res, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal query price response: %w", err)
	}
	return &resp, nil
}

// QueryTWAP queries the TWAP for a denom via ABCI.
func (c *QueryClient) QueryTWAP(_ context.Context, req *QueryTWAPRequest, _ ...grpc.CallOption) (*QueryTWAPResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	res, _, err := c.clientCtx.QueryWithData(
		fmt.Sprintf("custom/%s/twap/%s", ModuleName, req.Denom),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("query twap: %w", err)
	}
	var resp QueryTWAPResponse
	if err := json.Unmarshal(res, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal query twap response: %w", err)
	}
	return &resp, nil
}

// QueryAllPrices queries all price history for a denom via ABCI.
func (c *QueryClient) QueryAllPrices(_ context.Context, req *QueryAllPricesRequest, _ ...grpc.CallOption) (*QueryAllPricesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	res, _, err := c.clientCtx.QueryWithData(
		fmt.Sprintf("custom/%s/prices/%s/%d", ModuleName, req.Denom, req.FromHeight),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("query all prices: %w", err)
	}
	var resp QueryAllPricesResponse
	if err := json.Unmarshal(res, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal query all prices response: %w", err)
	}
	return &resp, nil
}
