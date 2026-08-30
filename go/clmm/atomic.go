package clmm

import (
	"fmt"

	onchainSui "github.com/k4k3ru-hub/onchain/go/sui"
)

type FlashLoan struct {
	Config   onchainSui.Argument
	Pool     onchainSui.Argument
	BalanceA onchainSui.Argument
	BalanceB onchainSui.Argument
	Receipt  onchainSui.Argument
}

// AppendFlashLoan appends a Cetus flash-loan Move call to a programmable transaction.
//
// The returned balances and receipt are transaction arguments that callers can
// route through other DEX calls before appending repayment.
//
// Parameters:
//   - builder: Programmable transaction builder.
//   - deployment: Cetus deployment.
//   - pool: Cetus pool state.
//   - loanA: Whether to borrow coin A instead of coin B.
//   - amount: Borrow amount in atomic units.
//
// Returns:
//   - Flash-loan transaction arguments.
//   - Validation error.
//
// Version:
//   - 2026-08-30: Added.
func AppendFlashLoan(builder *onchainSui.ProgrammableTransactionBuilder, deployment Deployment, pool Pool, loanA bool, amount uint64) (FlashLoan, error) {
	if builder == nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: builder=null")
	}
	if err := deployment.Validate(); err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	if pool.Address.IsZero() || pool.InitialVersion == 0 || pool.CoinTypeA == "" || pool.CoinTypeB == "" {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: pool=invalid")
	}
	if amount == 0 {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: amount=empty")
	}
	configObject := deployment.GlobalConfig
	configObject.Mutable = false
	config, err := builder.Object(onchainSui.InputKindShared, configObject)
	if err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	poolArgument, err := builder.Object(onchainSui.InputKindShared, onchainSui.ObjectInput{Address: pool.Address, Version: pool.InitialVersion, Mutable: true})
	if err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	loanAArgument, err := builder.Pure(bcsBool(loanA))
	if err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	amountArgument, err := builder.Pure(bcsUint64(amount))
	if err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	result, err := builder.MoveCall(onchainSui.MoveCall{
		Package:       deployment.PublishedAt,
		Module:        deployment.PoolModule,
		Function:      "flash_loan",
		TypeArguments: []string{pool.CoinTypeA, pool.CoinTypeB},
		Arguments:     []onchainSui.Argument{config, poolArgument, loanAArgument, amountArgument},
	})
	if err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	balanceA, err := onchainSui.NestedResult(result, 0)
	if err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	balanceB, err := onchainSui.NestedResult(result, 1)
	if err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	receipt, err := onchainSui.NestedResult(result, 2)
	if err != nil {
		return FlashLoan{}, fmt.Errorf("failed to append cetus clmm flash loan: %w", err)
	}
	return FlashLoan{Config: config, Pool: poolArgument, BalanceA: balanceA, BalanceB: balanceB, Receipt: receipt}, nil
}

// AppendRepayFlashLoan appends repayment for a previously created Cetus flash loan.
//
// Parameters:
//   - builder: Programmable transaction builder.
//   - deployment: Cetus deployment.
//   - pool: Cetus pool state.
//   - loan: Original flash-loan arguments.
//   - balanceA: Coin A balance after arbitrage routing.
//   - balanceB: Coin B balance after arbitrage routing.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-30: Added.
func AppendRepayFlashLoan(builder *onchainSui.ProgrammableTransactionBuilder, deployment Deployment, pool Pool, loan FlashLoan, balanceA, balanceB onchainSui.Argument) error {
	if builder == nil {
		return fmt.Errorf("failed to append cetus clmm flash loan repayment: builder=null")
	}
	if err := deployment.Validate(); err != nil {
		return fmt.Errorf("failed to append cetus clmm flash loan repayment: %w", err)
	}
	if pool.CoinTypeA == "" || pool.CoinTypeB == "" {
		return fmt.Errorf("failed to append cetus clmm flash loan repayment: pool=invalid")
	}
	_, err := builder.MoveCall(onchainSui.MoveCall{
		Package:       deployment.PublishedAt,
		Module:        deployment.PoolModule,
		Function:      "repay_flash_loan",
		TypeArguments: []string{pool.CoinTypeA, pool.CoinTypeB},
		Arguments:     []onchainSui.Argument{loan.Config, loan.Pool, balanceA, balanceB, loan.Receipt},
	})
	if err != nil {
		return fmt.Errorf("failed to append cetus clmm flash loan repayment: %w", err)
	}
	return nil
}
