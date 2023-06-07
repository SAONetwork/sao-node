package chain

import (
	"context"
	"fmt"
	loantypes "github.com/SaoNetwork/sao/x/loan/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"sao-node/types"
)

func (c *ChainSvc) Deposit(ctx context.Context, creator string, amount sdktypes.Coin) (string, error) {

	msg := &saotypes.MsgDeposit{
		Creator: creator,
		Amount:  amount,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgDeposit tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) Withdraw(ctx context.Context, creator string, amount sdktypes.Coin) (string, error) {

	msg := &saotypes.MsgWithdraw{
		Creator: creator,
		Amount:  amount,
	}
	resultChan := make(chan BroadcastTxJobResult)
	c.broadcastMsg(creator, msg, resultChan)
	result := <-resultChan
	if result.err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, result.err)
	}
	if result.resp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgWithdraw tx hash=%s, code=%d", result.resp.TxResponse.TxHash, result.resp.TxResponse.Code)
	}
	return result.resp.TxResponse.TxHash, nil
}

func (c *ChainSvc) GetCredit(ctx context.Context, account string) (sdktypes.Coin, error) {
	resp, err := c.loanClient.Credit(ctx, &loantypes.QueryGetCreditRequest{
		Account: account,
	})
	if err != nil {
		fmt.Println("account:", account, err)
		return sdktypes.Coin{}, types.Wrap(types.ErrQueryNodeFailed, err)
	}
	return resp.Amount, nil
}
