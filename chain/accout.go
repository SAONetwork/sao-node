package chain

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
)

func newAccountRegistry(ctx context.Context, chainId string) (cosmosaccount.Registry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return cosmosaccount.Registry{}, err
	}
	homePath := filepath.Join(home, "."+chainId)

	return cosmosaccount.New(
		cosmosaccount.WithKeyringBackend(cosmosaccount.KeyringTest),
		cosmosaccount.WithHome(homePath),
	)
}

func List(ctx context.Context, chainId string) error {
	accountRegistry, err := newAccountRegistry(ctx, chainId)
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
	}

	return nil
}

func Create(ctx context.Context, chainId string, name string) error {
	fmt.Println("ChainId: ", chainId)
	accountRegistry, err := newAccountRegistry(ctx, chainId)
	if err != nil {
		return err
	}

	account, mnemonic, err := accountRegistry.Create(name)
	if err != nil {
		return err
	}

	address, err := account.Address("cosmos")
	if err != nil {
		return err
	}
	fmt.Println("Account:", account.Name)
	fmt.Println("Address:", address)
	fmt.Println("Mnemonic:")
	fmt.Println(mnemonic)

	return nil
}

func Import(ctx context.Context, chainId string, name string, secret string, passphrase string) error {
	accountRegistry, err := newAccountRegistry(ctx, chainId)
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

func Export(ctx context.Context, chainId string, name string, passphrase string) error {
	accountRegistry, err := newAccountRegistry(ctx, chainId)
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
