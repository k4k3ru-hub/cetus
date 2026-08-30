package clmm

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	onchainSui "github.com/k4k3ru-hub/onchain/go/sui"
)

type Simulator interface {
	SimulateTransaction(context.Context, onchainSui.SimulationRequest) (*onchainSui.SimulationResult, error)
}

type QuoteExactInputParams struct {
	Sender   onchainSui.Address
	Pool     Pool
	AmountIn uint64
	A2B      bool
}

type QuoteResult struct {
	AmountIn       uint64
	AmountOut      uint64
	FeeAmount      uint64
	FeeRate        uint64
	AfterSqrtPrice *big.Int
	IsExceed       bool
}

type Quoter struct {
	deployment Deployment
	simulator  Simulator
}

// NewQuoter creates a Cetus CLMM simulation-backed quoter.
//
// Parameters:
//   - deployment: Cetus deployment.
//   - simulator: Sui transaction simulator.
//
// Returns:
//   - Quoter.
//   - Construction error.
//
// Version:
//   - 2026-08-30: Added.
func NewQuoter(deployment Deployment, simulator Simulator) (*Quoter, error) {
	if err := deployment.Validate(); err != nil {
		return nil, fmt.Errorf("failed to create cetus clmm quoter: %w", err)
	}
	if simulator == nil {
		return nil, fmt.Errorf("failed to create cetus clmm quoter: simulator=null")
	}
	return &Quoter{deployment: deployment, simulator: simulator}, nil
}

// QuoteExactInput simulates Cetus calculate_swap_result for one exact-input swap.
//
// Parameters:
//   - ctx: Request context.
//   - params: Quote parameters.
//
// Returns:
//   - Quote result.
//   - Quote or simulation error.
//
// Version:
//   - 2026-08-30: Added.
func (q *Quoter) QuoteExactInput(ctx context.Context, params QuoteExactInputParams) (QuoteResult, error) {
	if q == nil || q.simulator == nil {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: quoter=null")
	}
	if params.Sender.IsZero() {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: sender=empty")
	}
	if params.Pool.Address.IsZero() || params.Pool.InitialVersion == 0 {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: pool=invalid")
	}
	if params.AmountIn == 0 {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: amount_in=empty")
	}
	builder := onchainSui.NewProgrammableTransactionBuilder()
	pool, err := builder.Object(onchainSui.InputKindShared, onchainSui.ObjectInput{Address: params.Pool.Address, Version: params.Pool.InitialVersion})
	if err != nil {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: %w", err)
	}
	a2b, _ := builder.Pure(bcsBool(params.A2B))
	byAmountIn, _ := builder.Pure(bcsBool(true))
	amount, _ := builder.Pure(bcsUint64(params.AmountIn))
	_, err = builder.MoveCall(onchainSui.MoveCall{
		Package: q.deployment.PublishedAt, Module: q.deployment.FetcherModule, Function: "calculate_swap_result",
		TypeArguments: []string{params.Pool.CoinTypeA, params.Pool.CoinTypeB}, Arguments: []onchainSui.Argument{pool, a2b, byAmountIn, amount},
	})
	if err != nil {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: %w", err)
	}
	transaction, err := builder.Build()
	if err != nil {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: %w", err)
	}
	simulation, err := q.simulator.SimulateTransaction(ctx, onchainSui.SimulationRequest{Sender: params.Sender, Transaction: transaction})
	if err != nil {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: %w", err)
	}
	if len(simulation.CommandResults) != 1 || len(simulation.CommandResults[0].ReturnValues) == 0 {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: command_result=invalid")
	}
	result, err := parseCalculatedSwapResult(simulation.CommandResults[0].ReturnValues[0].BCS)
	if err != nil {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: %w", err)
	}
	if result.AmountIn == 0 || result.AmountOut == 0 || result.IsExceed {
		return QuoteResult{}, fmt.Errorf("failed to quote cetus clmm exact input: quote=invalid")
	}
	return result, nil
}

func parseCalculatedSwapResult(value []byte) (QuoteResult, error) {
	const fixedLength = 8*4 + 16 + 1
	if len(value) < fixedLength {
		return QuoteResult{}, fmt.Errorf("failed to parse cetus calculated swap result: bcs=too_short actual_length=%d min_length=%d", len(value), fixedLength)
	}
	result := QuoteResult{
		AmountIn: binary.LittleEndian.Uint64(value[0:8]), AmountOut: binary.LittleEndian.Uint64(value[8:16]),
		FeeAmount: binary.LittleEndian.Uint64(value[16:24]), FeeRate: binary.LittleEndian.Uint64(value[24:32]),
		AfterSqrtPrice: littleEndianUint128(value[32:48]), IsExceed: value[48] != 0,
	}
	return result, nil
}

func bcsBool(value bool) []byte {
	if value {
		return []byte{1}
	}
	return []byte{0}
}

func bcsUint64(value uint64) []byte {
	result := make([]byte, 8)
	binary.LittleEndian.PutUint64(result, value)
	return result
}

func littleEndianUint128(value []byte) *big.Int {
	reversed := append([]byte(nil), value...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return new(big.Int).SetBytes(reversed)
}
