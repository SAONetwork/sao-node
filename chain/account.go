package chain

import (
	"context"
	"fmt"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/mitchellh/go-homedir"
)

const DENOM string = "stake"

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
		return "", err
	}

	account, err := accountRegistry.GetByName(name)
	if err != nil {
		return "", err
	}
	address, err := account.Address("cosmos")
	if err != nil {
		return "", err
	}
	return address, nil
}

func SignByAccount(ctx context.Context, repo string, name string, payload []byte) ([]byte, error) {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return nil, err
	}

	sig, _, err := accountRegistry.Keyring.Sign(name, payload)
	if err != nil {
		return nil, err
	}

	return sig, err
}

func (c *ChainSvc) List(ctx context.Context, repo string) error {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return err
	}

	accounts, err := accountRegistry.List()
	if err != nil {
		return err
	}

	for _, account := range accounts {
		address, err := account.Address("cosmos")
		if err != nil {
			return err
		}

		fmt.Println("Account:", account.Name)
		fmt.Println("Address:", address)

		resp, err := c.bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
			Address: address,
			Denom:   DENOM,
		})
		if err != nil {
			return err
		}
		fmt.Println("Balance:", resp.Balance.Amount.Uint64(), DENOM)
	}

	return nil
}

func Create(ctx context.Context, repo string, name string) (string, string, string, error) {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return "", "", "", err
	}

	account, mnemonic, err := accountRegistry.Create(name)
	if err != nil {
		return "", "", "", err
	}

	address, err := account.Address("cosmos")
	if err != nil {
		return "", "", "", err
	}

	return account.Name, address, mnemonic, nil
}

func Import(ctx context.Context, repo string, name string, secret string, passphrase string) error {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return err
	}

	account, err := accountRegistry.Import(name, secret, passphrase)
	if err != nil {
		return err
	}

	address, err := account.Address("cosmos")
	if err != nil {
		return err
	}
	fmt.Println("Account:", account.Name)
	fmt.Println("Address:", address)

	return nil
}

func Export(ctx context.Context, repo string, name string, passphrase string) error {
	accountRegistry, err := newAccountRegistry(ctx, repo)
	if err != nil {
		return err
	}

	account, err := accountRegistry.GetByName(name)
	if err != nil {
		return err
	}
	address, err := account.Address("cosmos")
	if err != nil {
		return err
	}

	key, err := accountRegistry.Export(name, passphrase)
	if err != nil {
		return err
	}

	fmt.Println("Account:", name)
	fmt.Println("Address:", address)
	fmt.Println("Secret:")
	fmt.Println(key)

	return nil
}
