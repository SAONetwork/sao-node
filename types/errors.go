package types

import "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ModuleCommon        = "common"
	ErrInvalidRepoPath  = errors.Register(ModuleCommon, 10000, "invalid repo path")
	ErrCreateDirFailed  = errors.Register(ModuleCommon, 10001, "failed to create the directory")
	ErrCreateFileFailed = errors.Register(ModuleCommon, 10002, "failed to create the file")
	ErrOpenFileFailed   = errors.Register(ModuleCommon, 10003, "failed to create the file")
	ErrCloseFileFailed  = errors.Register(ModuleCommon, 10004, "failed to close the file")

	ErrInvalidBinaryName = errors.Register(ModuleCommon, 10005, "invalid binary name")

	ErrMarshalFailed   = errors.Register(ModuleChain, 10006, "failed to marshal payload")
	ErrUnMarshalFailed = errors.Register(ModuleChain, 10007, "failed to unmarshal JpayloadS")
)

var (
	ModuleChain = "chain"

	ErrCreateChainServiceFailed = errors.Register(ModuleChain, 11000, "failed to create the chain service")
	ErrStopChainServiceFailed   = errors.Register(ModuleChain, 11001, "failed to stop the chain service")

	ErrCreateAccountRegistryFailed = errors.Register(ModuleChain, 11002, "failed to create the account registry")

	ErrAccountNotFound     = errors.Register(ModuleChain, 11003, "account not found, check the keyring please")
	ErrListAccountsFailed  = errors.Register(ModuleChain, 11004, "failed to list the local accounts")
	ErrCreateAccountFailed = errors.Register(ModuleChain, 11005, "failed to create the account")
	ErrImportAccountFailed = errors.Register(ModuleChain, 11006, "failed to import the account")
	ErrExportAccountFailed = errors.Register(ModuleChain, 10007, "failed to export the account")

	ErrGetAddressFailed = errors.Register(ModuleChain, 11008, "failed to get address")
	ErrGetBalanceFailed = errors.Register(ModuleChain, 11009, "failed to get the balance")
	ErrSignedFailed     = errors.Register(ModuleChain, 11010, "failed to sign the payload")

	ErrTxCreateFailed  = errors.Register(ModuleChain, 11011, "failed to create the tx")
	ErrTxProcessFailed = errors.Register(ModuleChain, 11012, "failed to process the tx")
	ErrTxQueryFailed   = errors.Register(ModuleChain, 11013, "failed to query the tx")

	ErrGetSidDocumentFailed = errors.Register(ModuleChain, 11014, "failed to get the sid document")
	ErrQueryMetadataFailed  = errors.Register(ModuleChain, 11015, "failed to query the meta data")
	ErrQueryNodeFailed      = errors.Register(ModuleChain, 11016, "failed to query the node information")
	ErrQueryOrderFailed     = errors.Register(ModuleChain, 11017, "failed to query the order information")

	ErrInvalidSecrect       = errors.Register(ModuleChain, 11018, "invalid secrect")
	ErrInvalidPassphrase    = errors.Register(ModuleChain, 11019, "invalid passphrase")
	ErrCreateProviderFailed = errors.Register(ModuleChain, 11020, "failed to create the provider")
	ErrAuthenticateFailed   = errors.Register(ModuleChain, 11021, "failed to authenticate")
	ErrInvalidChainAddress  = errors.Register(ModuleChain, 11022, "invalid chain address")
	ErrCreateJwsFailed      = errors.Register(ModuleChain, 11023, "failed to create JWS")
	ErrMarshalJwsFailed     = errors.Register(ModuleChain, 11024, "failed to marshal JWS")
)

var (
	ModuleClient = "client"

	ErrEncodeConfigFailed     = errors.Register(ModuleClient, 12005, "failed to encode the configurations")
	ErrDecodeConfigFailed     = errors.Register(ModuleClient, 12006, "failed to decode the configurations")
	ErrWriteConfigFailed      = errors.Register(ModuleClient, 12007, "failed to write the configurations into file")
	ErrReadConfigFailed       = errors.Register(ModuleClient, 12008, "failed to read the configurations into file")
	ErrInvalidGateway         = errors.Register(ModuleClient, 12009, "invalid gateway")
	ErrInvalidToken           = errors.Register(ModuleClient, 12010, "invalid token")
	ErrCreateApiServiceFailed = errors.Register(ModuleClient, 12011, "failed to create API service")
	ErrGenerateDocFailed      = errors.Register(ModuleClient, 12012, "failed to generate the documents")
)

var (
	ModuleStore = "store"

	ErrOpenIpfsBackendFailed      = errors.Register(ModuleClient, 13000, "failed to open IPFS backend")
	ErrCreateIpfsApiServiceFailed = errors.Register(ModuleClient, 13001, "failed to create IPFS API service backend")
	ErrStoreFailed                = errors.Register(ModuleClient, 13002, "store data failed")
	ErrStatFailed                 = errors.Register(ModuleClient, 13003, "stat data failed")
	ErrGetFailed                  = errors.Register(ModuleClient, 13004, "get data failed")
	ErrInvalidPath                = errors.Register(ModuleClient, 13005, "invalid path")
	ErrLoadPluginsFailed          = errors.Register(ModuleClient, 13006, "failed to load plugins")
	ErrInitPluginsFailed          = errors.Register(ModuleClient, 13007, "failed to initializ plugins")
	ErrInjectPluginsFailed        = errors.Register(ModuleClient, 13008, "failed to inject plugins")
	ErrInitIpfsDaemonFailed       = errors.Register(ModuleClient, 13009, "failed to initializ IPFS daemon")
	ErrInitIpfsRepoFailed         = errors.Register(ModuleClient, 13010, "failed to initializ IPFS repo")
	ErrOpenRepoFailed             = errors.Register(ModuleClient, 13011, "failed to open IPFS repo")
	ErrUnSupportProtocol          = errors.Register(ModuleStore, 13012, "unsupported ipfs connection protocol")
)

var (
	ModuleModel = "model"

	ErrCreatePatchFailed = errors.Register(ModuleChain, 14000, "failed to create the patch")
	ErrDecodePatchFailed = errors.Register(ModuleChain, 14001, "failed to decode the patch")
	ErrApplyPatchFailed  = errors.Register(ModuleChain, 14002, "failed to apply the patch")

	ErrCalculateCidFailed = errors.Register(ModuleChain, 14003, "failed to calculate cid")
)

func Wrap(err0 error, err1 error) error {
	return errors.Wrapf(err0, "due to %w", err1)
}

func Wrapf(err error, format string, args ...interface{}) error {
	return errors.Wrapf(err, format, args...)
}
