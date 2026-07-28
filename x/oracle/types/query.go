package types

import "context"

// QueryServer defines the oracle gRPC-style query service interface.
type QueryServer interface {
	// QueryPrice queries the latest price for a given denom.
	QueryPrice(context.Context, *QueryPriceRequest) (*QueryPriceResponse, error)
	// QueryTWAP queries the latest computed TWAP for a given denom.
	QueryTWAP(context.Context, *QueryTWAPRequest) (*QueryTWAPResponse, error)
	// QueryAllPrices queries all stored price entries for a given denom.
	QueryAllPrices(context.Context, *QueryAllPricesRequest) (*QueryAllPricesResponse, error)
}

// ─── Request / Response types ─────────────────────────────────────────────────

// QueryPriceRequest is the request type for the QueryPrice RPC.
type QueryPriceRequest struct {
	Denom string `json:"denom"`
}

func (r *QueryPriceRequest) ProtoMessage()  {}
func (r *QueryPriceRequest) Reset()         {}
func (r *QueryPriceRequest) String() string { return r.Denom }

// QueryPriceResponse is the response type for the QueryPrice RPC.
type QueryPriceResponse struct {
	Price PriceEntry `json:"price"`
}

func (r *QueryPriceResponse) ProtoMessage()  {}
func (r *QueryPriceResponse) Reset()         {}
func (r *QueryPriceResponse) String() string { return "" }

// QueryTWAPRequest is the request type for the QueryTWAP RPC.
type QueryTWAPRequest struct {
	Denom string `json:"denom"`
}

func (r *QueryTWAPRequest) ProtoMessage()  {}
func (r *QueryTWAPRequest) Reset()         {}
func (r *QueryTWAPRequest) String() string { return r.Denom }

// QueryTWAPResponse is the response type for the QueryTWAP RPC.
type QueryTWAPResponse struct {
	TWAP TWAPData `json:"twap"`
}

func (r *QueryTWAPResponse) ProtoMessage()  {}
func (r *QueryTWAPResponse) Reset()         {}
func (r *QueryTWAPResponse) String() string { return "" }

// QueryAllPricesRequest is the request type for the QueryAllPrices RPC.
type QueryAllPricesRequest struct {
	Denom      string `json:"denom"`
	FromHeight int64  `json:"from_height"`
}

func (r *QueryAllPricesRequest) ProtoMessage()  {}
func (r *QueryAllPricesRequest) Reset()         {}
func (r *QueryAllPricesRequest) String() string { return r.Denom }

// QueryAllPricesResponse is the response type for the QueryAllPrices RPC.
type QueryAllPricesResponse struct {
	Prices []PriceEntry `json:"prices"`
}

func (r *QueryAllPricesResponse) ProtoMessage()  {}
func (r *QueryAllPricesResponse) Reset()         {}
func (r *QueryAllPricesResponse) String() string { return "" }
