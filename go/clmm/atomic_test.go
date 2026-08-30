package clmm

import (
	"testing"

	onchainSui "github.com/k4k3ru-hub/onchain/go/sui"
)

func TestAppendFlashLoanAndRepayment(t *testing.T) {
	poolAddress, _ := onchainSui.ParseAddress("0x9")
	pool := Pool{Address: poolAddress, InitialVersion: 3, CoinTypeA: "0x2::sui::SUI", CoinTypeB: "0x3::usdc::USDC"}
	builder := onchainSui.NewProgrammableTransactionBuilder()
	loan, err := AppendFlashLoan(builder, testDeployment(), pool, true, 100)
	if err != nil {
		t.Fatalf("AppendFlashLoan() returned an unexpected error: %v", err)
	}
	if err := AppendRepayFlashLoan(builder, testDeployment(), pool, loan, loan.BalanceA, loan.BalanceB); err != nil {
		t.Fatalf("AppendRepayFlashLoan() returned an unexpected error: %v", err)
	}
	transaction, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() returned an unexpected error: %v", err)
	}
	if len(transaction.Commands) != 2 || transaction.Commands[0].MoveCall == nil || transaction.Commands[0].MoveCall.Function != "flash_loan" || transaction.Commands[1].MoveCall == nil || transaction.Commands[1].MoveCall.Function != "repay_flash_loan" {
		t.Fatalf("transaction commands = %+v", transaction.Commands)
	}
	if !transaction.Inputs[1].Object.Mutable {
		t.Fatalf("pool input is not mutable: %+v", transaction.Inputs[1])
	}
}
