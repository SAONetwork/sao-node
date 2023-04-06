package chain

import (
	"context"
	"fmt"
	"math/big"
	"sao-node/types"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/go-bip39"

	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/mitchellh/go-homedir"
)

const DENOM string = "sao"

func newAccountRegistry(_ context.Context, repo string) (cosmosaccount.Registry, error) {
	repoPath, err := homedir.Expand(repo)
	if err != nil {
		return cosmosaccount.Registry{}, err
	}

	return cosmosaccount.New(
		cosmosaccount.WithKeyringBackend(cosmosaccount.KeyringTest),
		cosmosaccount.WithHome(repoPath),
	)
}

func GetAddress(ctx context.Context, repo string, name string) (string, error) {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return "", types.Wrap(types.ErrGetAddressFailed, err)
	}

	account, err := accountRegistry.GetByName(name)
	if err != nil {
		return "", types.Wrap(types.ErrGetAddressFailed, err)
	}
	address, err := account.Address(ADDRESS_PREFIX)
	if err != nil {
		return "", types.Wrap(types.ErrGetAddressFailed, err)
	}
	return address, nil
}

func SignByAccount(ctx context.Context, repo string, name string, payload []byte) ([]byte, error) {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return nil, types.Wrap(types.ErrSignedFailed, err)
	}

	sig, _, err := accountRegistry.Keyring.Sign(name, payload)
	if err != nil {
		return nil, types.Wrap(types.ErrSignedFailed, err)
	}

	return sig, nil
}

func SignByAddress(ctx context.Context, repo string, address string, payload []byte) ([]byte, error) {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return nil, types.Wrap(types.ErrSignedFailed, err)
	}

	addr, err := sdktypes.AccAddressFromBech32(address)
	if err != nil {
		return nil, types.Wrap(types.ErrSignedFailed, err)
	}

	sig, _, err := accountRegistry.Keyring.SignByAddress(addr, payload)
	if err != nil {
		return nil, types.Wrap(types.ErrSignedFailed, err)
	}

	return sig, nil
}

func (c *ChainSvc) List(ctx context.Context, repo string) error {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return types.Wrap(types.ErrListAccountsFailed, err)
	}

	accounts, err := accountRegistry.List()
	if err != nil {
		return types.Wrap(types.ErrListAccountsFailed, err)
	}

	if len(accounts) > 0 {
		fmt.Println("======================================================")
	}

	for _, account := range accounts {
		address, err := account.Address(ADDRESS_PREFIX)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		fmt.Println("Account:", account.Name)
		fmt.Println("Address:", address)

		resp, err := c.bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
			Address: address,
			Denom:   DENOM,
		})
		if err != nil {
			return types.Wrap(types.ErrGetBalanceFailed, err)
		}
		fmt.Println("Balance:", resp.Balance.Amount, DENOM)
		fmt.Println("======================================================")
	}

	return nil
}

func (c *ChainSvc) ShowBalance(ctx context.Context, address string) {
	fmt.Println("Address:", address)

	resp, err := c.bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   DENOM,
	})
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Println("Balance:", resp.Balance.Amount, DENOM)
}

func (c *ChainSvc) Send(ctx context.Context, from string, to string, amount int64) (string, error) {
	signerAcc, err := c.cosmos.Account(from)
	if err != nil {
		return "", types.Wrap(types.ErrAccountNotFound, err)
	}

	tx, err := c.cosmos.BankSendTx(ctx, signerAcc, to, append(make(sdktypes.Coins, 0), sdktypes.Coin{
		Denom:  DENOM,
		Amount: sdktypes.NewIntFromBigInt(big.NewInt(amount)),
	}))
	if err != nil {
		return "", types.Wrap(types.ErrTxCreateFailed, err)
	}

	txResp, err := tx.Broadcast(ctx)
	if err != nil {
		return "", types.Wrap(types.ErrTxProcessFailed, err)
	}
	if txResp.TxResponse.Code != 0 {
		return "", types.Wrapf(types.ErrTxProcessFailed, "MsgStore tx hash=%s, code=%d", txResp.TxResponse.TxHash, txResp.TxResponse.Code)
	}

	return txResp.TxResponse.TxHash, nil
}

func GenerateMnemonic(ctx context.Context) (string, error) {
	entropySeed, err := bip39.NewEntropy(256)
	if err != nil {
		return "", types.Wrap(types.ErrGenerateMnemonicFailed, err)
	}
	mnemonic, err := bip39.NewMnemonic(entropySeed)
	if err != nil {
		return "", types.Wrap(types.ErrGenerateMnemonicFailed, err)
	}
	return mnemonic, nil
}

func GenerateAccount(ctx context.Context, repo string, name string, mnemonic string) (string, error) {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return "", types.Wrap(types.ErrCreateAccountFailed, err)
	}

	hdPath := hd.CreateHDPath(sdktypes.GetConfig().GetCoinType(), 0, 0).String()
	algos, _ := accountRegistry.Keyring.SupportedAlgorithms()
	if err != nil {
		return "", types.Wrap(types.ErrCreateAccountFailed, err)
	}
	algo, err := keyring.NewSigningAlgoFromString(string(hd.Secp256k1Type), algos)
	if err != nil {
		return "", types.Wrap(types.ErrCreateAccountFailed, err)
	}
	record, err := accountRegistry.Keyring.NewAccount(name, mnemonic, "", hdPath, algo)
	if err != nil {
		return "", types.Wrap(types.ErrCreateAccountFailed, err)
	}
	account := cosmosaccount.Account{
		Name:   name,
		Record: record,
	}

	address, err := account.Address(ADDRESS_PREFIX)
	if err != nil {
		return "", types.Wrap(types.ErrCreateAccountFailed, err)
	}

	return address, nil
}

func Create(ctx context.Context, repo string, name string) (string, string, string, error) {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return "", "", "", types.Wrap(types.ErrCreateAccountFailed, err)
	}

	account, mnemonic, err := accountRegistry.Create(name)
	if err != nil {
		return "", "", "", types.Wrap(types.ErrCreateAccountFailed, err)
	}

	address, err := account.Address(ADDRESS_PREFIX)
	if err != nil {
		return "", "", "", types.Wrap(types.ErrCreateAccountFailed, err)
	}

	return account.Name, address, mnemonic, nil
}

func Import(ctx context.Context, repo string, name string, secret string, passphrase string) error {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return types.Wrap(types.ErrImportAccountFailed, err)
	}

	account, err := accountRegistry.Import(name, secret, passphrase)
	if err != nil {
		return types.Wrap(types.ErrImportAccountFailed, err)
	}

	address, err := account.Address(ADDRESS_PREFIX)
	if err != nil {
		return types.Wrap(types.ErrImportAccountFailed, err)
	}
	fmt.Println("Account:", account.Name)
	fmt.Println("Address:", address)

	return nil
}

func Export(ctx context.Context, repo string, name string, passphrase string) error {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return types.Wrap(types.ErrExportAccountFailed, err)
	}

	account, err := accountRegistry.GetByName(name)
	if err != nil {
		return types.Wrap(types.ErrExportAccountFailed, err)
	}
	address, err := account.Address(ADDRESS_PREFIX)
	if err != nil {
		return types.Wrap(types.ErrExportAccountFailed, err)
	}

	key, err := accountRegistry.Export(name, passphrase)
	if err != nil {
		return types.Wrap(types.ErrExportAccountFailed, err)
	}

	fmt.Println("Account:", name)
	fmt.Println("Address:", address)
	fmt.Println("Secret:")
	fmt.Println(key)

	return nil
}
