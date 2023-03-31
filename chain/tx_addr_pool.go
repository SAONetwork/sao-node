package chain

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type AddressPool struct {
	ctx         context.Context
	Addresses   map[string]string
	RootAddress string
	keyringHome string
	aplk        sync.Mutex
}

func CreateAddressPool(ctx context.Context, keyringHome string, size uint) error {
	rootAddress, err := GetAddress(ctx, keyringHome, "tx_addr_0")
	if err == nil {
		log.Infof("there is a root address[%s] with name tx_addr_0 already", rootAddress)
	} else {
		mnemonic, err := GenerateMnemonic(ctx)
		if err != nil {
			return err
		}
		log.Infof("YOUR MNEMONIC: %s", mnemonic)

		rootAddress, err = generateAddressFromMnemonic(ctx, keyringHome, mnemonic, 0)
		if err != nil {
			return err
		}
		log.Infof("pool address[tx_addr_0]: %s", rootAddress)
	}

	for i := uint(1); i < size; i++ {
		name := fmt.Sprintf("tx_addr_%d", i)
		address, err := GetAddress(ctx, keyringHome, name)
		if err == nil {
			log.Infof("there is a pool address[%s] with name %s already", address, name)
		} else {
			mnemonic, err := GenerateMnemonic(ctx)
			if err != nil {
				return err
			}
			log.Infof("YOUR MNEMONIC: %s", mnemonic)

			address, err = generateAddressFromMnemonic(ctx, keyringHome, mnemonic, i)
			if err != nil {
				return err
			}
			log.Infof("pool address[%s]: %s", name, address)
		}
	}

	return nil
}

func LoadAddressPool(ctx context.Context, keyringHome string, size uint) (*AddressPool, error) {
	pool := &AddressPool{
		ctx:         ctx,
		Addresses:   make(map[string]string, size),
		keyringHome: keyringHome,
	}

	rootAddress, err := GetAddress(ctx, keyringHome, "tx_addr_0")
	if err != nil {
		return nil, err
	}
	pool.RootAddress = rootAddress
	pool.Addresses[rootAddress] = "Available"

	for i := uint(1); i < size; i++ {
		name := fmt.Sprintf("tx_addr_%d", i)
		address, err := GetAddress(ctx, keyringHome, name)
		if err != nil {
			log.Error(err)
			continue
		}
		pool.Addresses[address] = "Available"
	}

	return pool, nil
}

func (ap *AddressPool) SetRootAddress(address string) {
	ap.aplk.Lock()
	defer ap.aplk.Unlock()

	ap.RootAddress = address
	ap.Addresses[address] = "root"
}

func generateAddressFromMnemonic(ctx context.Context, keyringHome, mnemonic string, index uint) (string, error) {
	name := fmt.Sprintf("tx_addr_%d", index)
	return GenerateAccount(ctx, keyringHome, name, mnemonic)
}

func (p *AddressPool) GetRandomAddress(ctx context.Context) (string, error) {
	p.aplk.Lock()
	defer p.aplk.Unlock()

	if len(p.Addresses) == 0 {
		return "", errors.New("address pool is empty")
	}

	availableAddresses := make([]string, 0, len(p.Addresses))
	for address, status := range p.Addresses {
		if status == "Available" {
			availableAddresses = append(availableAddresses, address)
		}
	}

	if len(availableAddresses) == 0 {
		return "", errors.New("no available addresses")
	}

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(availableAddresses))
	randomAddress := availableAddresses[randomIndex]

	p.Addresses[randomAddress] = "InUse"

	return randomAddress, nil
}

func (p *AddressPool) SetAddressAvailable(address string) error {
	p.aplk.Lock()
	defer p.aplk.Unlock()

	status, ok := p.Addresses[address]
	if !ok {
		return errors.New("address not found in pool")
	}

	if status != "InUse" {
		return errors.New("address is not in use")
	}

	p.Addresses[address] = "Available"

	return nil
}
