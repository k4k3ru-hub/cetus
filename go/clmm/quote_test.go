package clmm

import (
	"context"
	"encoding/binary"
	"testing"

	onchainSui "github.com/k4k3ru-hub/onchain/go/sui"
)

type quoteTestSimulator struct {
	request onchainSui.SimulationRequest
	result  *onchainSui.SimulationResult
}

func (s *quoteTestSimulator) SimulateTransaction(_ context.Context, request onchainSui.SimulationRequest) (*onchainSui.SimulationResult, error) {
	s.request = request
	return s.result, nil
}

func TestQuoteExactInput(t *testing.T) {
	packageAddress, _ := onchainSui.ParseAddress("0x1")
	fetcherPackage, _ := onchainSui.ParseAddress("0x5")
	configAddress, _ := onchainSui.ParseAddress("0x2")
	clockAddress, _ := onchainSui.ParseAddress("0x6")
	poolAddress, _ := onchainSui.ParseAddress("0x3")
	sender, _ := onchainSui.ParseAddress("0x4")
	value := make([]byte, 49)
	binary.LittleEndian.PutUint64(value[0:8], 100)
	binary.LittleEndian.PutUint64(value[8:16], 99)
	binary.LittleEndian.PutUint64(value[16:24], 1)
	binary.LittleEndian.PutUint64(value[24:32], 100)
	value[32] = 5
	simulator := &quoteTestSimulator{result: &onchainSui.SimulationResult{CommandResults: []onchainSui.SimulationCommandResult{{ReturnValues: []onchainSui.CommandOutput{{BCS: value}}}}}}
	quoter, err := NewQuoter(Deployment{
		Package: packageAddress, PublishedAt: packageAddress, FetcherPackage: fetcherPackage,
		GlobalConfig: onchainSui.ObjectInput{Address: configAddress, Version: 1}, Clock: onchainSui.ObjectInput{Address: clockAddress, Version: 1},
		PoolModule: "pool", FetcherModule: "fetcher_script",
	}, simulator)
	if err != nil {
		t.Fatalf("NewQuoter() returned an unexpected error: %v", err)
	}
	result, err := quoter.QuoteExactInput(context.Background(), QuoteExactInputParams{Sender: sender, Pool: Pool{Address: poolAddress, InitialVersion: 2, CoinTypeA: "0x2::sui::SUI", CoinTypeB: "0x3::usdc::USDC"}, AmountIn: 100, A2B: true})
	if err != nil {
		t.Fatalf("QuoteExactInput() returned an unexpected error: %v", err)
	}
	if result.AmountOut != 99 || result.FeeAmount != 1 || len(simulator.request.Transaction.Commands) != 1 || simulator.request.Transaction.Commands[0].MoveCall == nil || simulator.request.Transaction.Commands[0].MoveCall.Package != fetcherPackage || simulator.request.Transaction.Commands[0].MoveCall.Function != "calculate_swap_result" {
		t.Fatalf("QuoteExactInput() = %+v request=%+v", result, simulator.request)
	}
}

func TestQuoteExactOutput(t *testing.T) {
	packageAddress, _ := onchainSui.ParseAddress("0x1")
	fetcherPackage, _ := onchainSui.ParseAddress("0x5")
	configAddress, _ := onchainSui.ParseAddress("0x2")
	clockAddress, _ := onchainSui.ParseAddress("0x6")
	poolAddress, _ := onchainSui.ParseAddress("0x3")
	sender, _ := onchainSui.ParseAddress("0x4")
	value := make([]byte, 49)
	binary.LittleEndian.PutUint64(value[0:8], 101)
	binary.LittleEndian.PutUint64(value[8:16], 100)
	binary.LittleEndian.PutUint64(value[16:24], 1)
	binary.LittleEndian.PutUint64(value[24:32], 100)
	simulator := &quoteTestSimulator{result: &onchainSui.SimulationResult{CommandResults: []onchainSui.SimulationCommandResult{{ReturnValues: []onchainSui.CommandOutput{{BCS: value}}}}}}
	quoter, err := NewQuoter(Deployment{
		Package: packageAddress, PublishedAt: packageAddress, FetcherPackage: fetcherPackage,
		GlobalConfig: onchainSui.ObjectInput{Address: configAddress, Version: 1}, Clock: onchainSui.ObjectInput{Address: clockAddress, Version: 1},
		PoolModule: "pool", FetcherModule: "fetcher_script",
	}, simulator)
	if err != nil {
		t.Fatalf("NewQuoter() returned an unexpected error: %v", err)
	}
	result, err := quoter.QuoteExactOutput(context.Background(), QuoteExactOutputParams{Sender: sender, Pool: Pool{Address: poolAddress, InitialVersion: 2, CoinTypeA: "0x2::sui::SUI", CoinTypeB: "0x3::usdc::USDC"}, AmountOut: 100, A2B: false})
	if err != nil {
		t.Fatalf("QuoteExactOutput() returned an unexpected error: %v", err)
	}
	if result.AmountIn != 101 || len(simulator.request.Transaction.Inputs) != 4 || len(simulator.request.Transaction.Inputs[2].Pure) != 1 || simulator.request.Transaction.Inputs[2].Pure[0] != 0 {
		t.Fatalf("QuoteExactOutput() = %+v request=%+v", result, simulator.request)
	}
}
