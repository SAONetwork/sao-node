package types

import "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ModuleChain = "chain"

	ErrCreateChainServiceFailed = errors.Register(ModuleChain, 10000, "failed to create the chain service")
	ErrStopChainServiceFailed   = errors.Register(ModuleChain, 10001, "failed to stop the chain service")

	ErrCreateAccountRegistryFailed = errors.Register(ModuleChain, 10002, "failed to create the account registry")

	ErrAccountNotFound     = errors.Register(ModuleChain, 10003, "account not found, check the keyring please")
	ErrListAccountsFailed  = errors.Register(ModuleChain, 10004, "failed to list the local accounts")
	ErrCreateAccountFailed = errors.Register(ModuleChain, 10005, "failed to create the account")
	ErrImportAccountFailed = errors.Register(ModuleChain, 10006, "failed to import the account")
	ErrExportAccountFailed = errors.Register(ModuleChain, 10007, "failed to export the account")

	ErrGetAddressFailed = errors.Register(ModuleChain, 10008, "failed to get address")
	ErrGetBalanceFailed = errors.Register(ModuleChain, 10009, "failed to get the balance")
	ErrSignedFailed     = errors.Register(ModuleChain, 10010, "failed to sign the payload")

	ErrTxCreateFailed  = errors.Register(ModuleChain, 10011, "failed to create the tx")
	ErrTxProcessFailed = errors.Register(ModuleChain, 10012, "failed to process the tx")
	ErrTxQueryFailed   = errors.Register(ModuleChain, 10013, "failed to query the tx")

	ErrGetSidDocumentFailed = errors.Register(ModuleChain, 10014, "failed to get the sid document")
	ErrQueryMetadataFailed  = errors.Register(ModuleChain, 10015, "failed to query the meta data")
	ErrQueryNodeFailed      = errors.Register(ModuleChain, 10016, "failed to query the node information")
	ErrQueryOrderFailed     = errors.Register(ModuleChain, 10017, "failed to query the order information")
)

func Wrap(err0 error, err1 error) error {
	return errors.Wrapf(err0, ", due to %w", err1)
}

func Wrapf(err error, format string, args ...interface{}) error {
	return errors.Wrapf(err, format, args...)
}
