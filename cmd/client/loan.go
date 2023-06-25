package main

import (
	"fmt"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
	saoclient "sao-node/client"
	cliutil "sao-node/cmd"
	"sao-node/types"
)

var loanCmd = &cli.Command{
	Name:  "loan",
	Usage: "loan management",
	Subcommands: []*cli.Command{
		loanDepositCmd,
		loanWithdrawCmd,
	},
}

var loanDepositCmd = &cli.Command{
	Name:  "deposit",
	Usage: "Deposit coins in the loan pool to earn loan income",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "creator",
			Usage:    "deposit address",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "amount",
			Usage:    "amount to deposit.",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		opt := saoclient.SaoClientOptions{
			Repo:        cctx.String(FlagClientRepo),
			Gateway:     "none",
			ChainAddr:   cliutil.ChainAddress,
			KeyringHome: cliutil.KeyringHome,
		}
		saoclient, closer, err := saoclient.NewSaoClient(cctx.Context, opt)
		if err != nil {
			return err
		}
		defer closer()

		creator := cctx.String("creator")

		balance, err := saoclient.GetBalance(ctx, creator)
		if err != nil {
			return err
		}

		amount := sdktypes.NewCoin("sao", sdktypes.NewInt(cctx.Int64("amount")))

		if balance.AmountOf("sao").LT(amount.Amount) {
			fmt.Println("insufficient funds, expected: ", amount.String(), ", have: ", balance.String())
			return types.ErrInsufficientFunds
		}

		tx, err := saoclient.Deposit(ctx, creator, amount)
		if err != nil {
			return err
		}

		fmt.Println(creator, " deposit ", amount.String())
		fmt.Println("txHash ", tx)

		return nil
	},
}

var loanWithdrawCmd = &cli.Command{
	Name:  "withdraw",
	Usage: "Withdraw coins from the loan pool",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "creator",
			Usage:    "withdraw address",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "amount",
			Usage:    "amount to withdraw.",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		opt := saoclient.SaoClientOptions{
			Repo:        cctx.String(FlagClientRepo),
			Gateway:     "none",
			ChainAddr:   cliutil.ChainAddress,
			KeyringHome: cliutil.KeyringHome,
		}
		saoclient, closer, err := saoclient.NewSaoClient(cctx.Context, opt)
		if err != nil {
			return err
		}
		defer closer()

		creator := cctx.String("creator")

		total, available, err := saoclient.GetAvailable(ctx, creator)
		if err != nil {
			return err
		}

		amount := sdktypes.NewCoin("sao", sdktypes.NewInt(cctx.Int64("amount")))

		if available.Amount.LT(amount.Amount) {
			fmt.Println("insufficient funds, expected: ", amount.String(), ", total: ", total.String(), ", available now: ", available.String())
			return types.ErrInsufficientFunds
		}

		tx, err := saoclient.Withdraw(ctx, creator, amount)
		if err != nil {
			return err
		}

		fmt.Println(creator, " withdraw ", amount.String())
		fmt.Println("txHash ", tx)

		return nil
	},
}
