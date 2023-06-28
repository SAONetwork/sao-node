package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ModuleCommon        = "common"
	ErrInvalidRepoPath  = errors.Register(ModuleCommon, 10000, "invalid repo path")
	ErrCreateDirFailed  = errors.Register(ModuleCommon, 10001, "failed to create the directory")
	ErrCreateFileFailed = errors.Register(ModuleCommon, 10002, "failed to create the file")
	ErrOpenFileFailed   = errors.Register(ModuleCommon, 10003, "failed to open the file")
	ErrReadFileFailed   = errors.Register(ModuleCommon, 10004, "failed to read the file")
	ErrWriteFileFailed  = errors.Register(ModuleCommon, 10005, "failed to write the file")
	ErrCloseFileFailed  = errors.Register(ModuleCommon, 10006, "failed to close the file")
	ErrInitRepoFailed   = errors.Register(ModuleCommon, 10007, "failed to initialize the repo")

	ErrInvalidBinaryName = errors.Register(ModuleCommon, 10008, "invalid binary name")

	ErrMarshalFailed   = errors.Register(ModuleCommon, 10009, "failed to marshal payload")
	ErrUnMarshalFailed = errors.Register(ModuleCommon, 10010, "failed to unmarshal payload")
	ErrUnSupport       = errors.Register(ModuleCommon, 10011, "not implemented yet")
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
	ErrExportAccountFailed = errors.Register(ModuleChain, 11007, "failed to export the account")

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
	ErrInvalidJwt           = errors.Register(ModuleChain, 11025, "invalid JWT")

	ErrQueryHeightFailed   = errors.Register(ModuleChain, 11026, "failed to query the latest height")
	ErrInconsistentAddress = errors.Register(ModuleChain, 11027, "inconsistent address")

	ErrGenerateMnemonicFailed = errors.Register(ModuleChain, 11028, "failed to generate mnemonic")

	ErrQueryPledgeFailed      = errors.Register(ModuleChain, 11030, "failed to query the pledge information")

	ErrInvalidValidator       = errors.Register(ModuleChain, 11031, "invalid validator")

	ErrQueryShardFailed = errors.Register(ModuleChain, 11032, "failed to query the shard information")

)

var (
	ModuleClient = "client"

	ErrInvalidConfig          = errors.Register(ModuleClient, 12004, "invalid configurations")
	ErrEncodeConfigFailed     = errors.Register(ModuleClient, 12005, "failed to encode the configurations")
	ErrDecodeConfigFailed     = errors.Register(ModuleClient, 12006, "failed to decode the configurations")
	ErrWriteConfigFailed      = errors.Register(ModuleClient, 12007, "failed to write the configurations into file")
	ErrReadConfigFailed       = errors.Register(ModuleClient, 12008, "failed to read the configurations into file")
	ErrInvalidGateway         = errors.Register(ModuleClient, 12009, "invalid gateway")
	ErrInvalidToken           = errors.Register(ModuleClient, 12010, "invalid token")
	ErrCreateApiServiceFailed = errors.Register(ModuleClient, 12011, "failed to create API service")
	ErrGenerateDocFailed      = errors.Register(ModuleClient, 12012, "failed to generate the documents")
	ErrOpenDataStoreFailed    = errors.Register(ModuleClient, 12013, "failed to open the data store")
	ErrInvalidParameters      = errors.Register(ModuleClient, 12014, "invalid parameters")
	ErrCreateClientFailed     = errors.Register(ModuleClient, 12015, "failed to create client")
)

var (
	ModuleStore = "store"

	ErrOpenIpfsBackendFailed      = errors.Register(ModuleStore, 13000, "failed to open IPFS backend")
	ErrCreateIpfsApiServiceFailed = errors.Register(ModuleStore, 13001, "failed to create IPFS API service backend")
	ErrStoreFailed                = errors.Register(ModuleStore, 13002, "store data failed")
	ErrStatFailed                 = errors.Register(ModuleStore, 13003, "stat data failed")
	ErrGetFailed                  = errors.Register(ModuleStore, 13004, "get data failed")
	ErrInvalidPath                = errors.Register(ModuleStore, 13005, "invalid path")
	ErrLoadPluginsFailed          = errors.Register(ModuleStore, 13006, "failed to load plugins")
	ErrInitPluginsFailed          = errors.Register(ModuleStore, 13007, "failed to initializ plugins")
	ErrInjectPluginsFailed        = errors.Register(ModuleStore, 13008, "failed to inject plugins")
	ErrInitIpfsDaemonFailed       = errors.Register(ModuleStore, 13009, "failed to initializ IPFS daemon")
	ErrInitIpfsRepoFailed         = errors.Register(ModuleStore, 13010, "failed to initializ IPFS repo")
	ErrOpenRepoFailed             = errors.Register(ModuleStore, 13011, "failed to open IPFS repo")
	ErrUnSupportProtocol          = errors.Register(ModuleStore, 13012, "unsupported ipfs connection protocol")
	ErrRemoveFailed               = errors.Register(ModuleStore, 13013, "remove data failed")
	ErrDataMissing                = errors.Register(ModuleStore, 13014, "cannot found the data")
)

var (
	ModuleModel = "model"

	ErrCreatePatchFailed = errors.Register(ModuleModel, 14000, "failed to create the patch")
	ErrDecodePatchFailed = errors.Register(ModuleModel, 14001, "failed to decode the patch")
	ErrApplyPatchFailed  = errors.Register(ModuleModel, 14002, "failed to apply the patch")

	ErrCalculateCidFailed = errors.Register(ModuleModel, 14003, "failed to calculate cid")

	ErrConflictName   = errors.Register(ModuleModel, 14004, "conflict name")
	ErrNotFound       = errors.Register(ModuleModel, 14005, "not found")
	ErrCacheGetFailed = errors.Register(ModuleModel, 14006, "failed to get value from cache")

	ErrInvalidDid       = errors.Register(ModuleModel, 14007, "invalid did")
	ErrInvalidSignature = errors.Register(ModuleModel, 14008, "invalid signature")

	ErrGenerateTokenFaild = errors.Register(ModuleModel, 14009, "failed to genrate the token")
	ErrGetHttpUrlFaild    = errors.Register(ModuleModel, 14010, "failed to get the HTTP URL")
	ErrGetIpfsUrlFaild    = errors.Register(ModuleModel, 14011, "failed to get the IPFS URL")

	ErrInvalidCommitInfo = errors.Register(ModuleModel, 14012, "invalid commit information")
	ErrInvalidCid        = errors.Register(ModuleModel, 14013, "invalid cid")
	ErrInvalidAlias      = errors.Register(ModuleModel, 14014, "invalid alias")

	ErrAddResourceFaild = errors.Register(ModuleModel, 14015, "failed to add the resource")
	ErrCompileFaild     = errors.Register(ModuleModel, 14016, "failed to compile")
	ErrAddRuleFaild     = errors.Register(ModuleModel, 14017, "failed to add the rule")
	ErrAddFactFaild     = errors.Register(ModuleModel, 14018, "failed to add the fact")
	ErrRuleExcuteFaild  = errors.Register(ModuleModel, 14019, "failed to excute the rule")
	ErrRuleCheckFaild   = errors.Register(ModuleModel, 14020, "failed to pass the rule check")
	ErrInvalidRule      = errors.Register(ModuleModel, 14021, "invlaid rule")
	ErrSchemaCheckFaild = errors.Register(ModuleModel, 14022, "failed to pass the schema check")

	ErrInvalidVersion     = errors.Register(ModuleModel, 14023, "invalid version")
	ErrInvalidDataId      = errors.Register(ModuleModel, 14024, "invalid dataId")
	ErrConflictId         = errors.Register(ModuleModel, 14025, "conflict dataId or alias")
	ErrInvalidContent     = errors.Register(ModuleModel, 14026, "invalid content")
	ErrInvalidSchema      = errors.Register(ModuleModel, 14027, "invalid schema")
	ErrProcessOrderFailed = errors.Register(ModuleModel, 14028, "failed to process the order")
	ErrExpiredOrder       = errors.Register(ModuleModel, 14029, "expired order")
	ErrRetriesExceed      = errors.Register(ModuleModel, 14030, "shard retries too many times")
)

var (
	ModuleNetwork = "network"

	ErrCreateP2PServiceFaild      = errors.Register(ModuleChain, 15000, "failed to create the P2P service")
	ErrStartLibP2PRPCServerFailed = errors.Register(ModuleNetwork, 15001, "failed to start libp2p RPC server")
	ErrInvalidServerAddress       = errors.Register(ModuleNetwork, 15002, "invalid transport server address")
	ErrStartPRPCServerFailed      = errors.Register(ModuleNetwork, 15003, "failed to start RPC server")
	ErrConnectFailed              = errors.Register(ModuleNetwork, 15004, "failed to connect")
	ErrCreateStreamFailed         = errors.Register(ModuleNetwork, 15005, "failed to create the stream")
	ErrCloseStreamFailed          = errors.Register(ModuleNetwork, 15006, "failed to close the stream")
	ErrSendRequestFailed          = errors.Register(ModuleNetwork, 15007, "failed to send the request")
	ErrReadResponseFailed         = errors.Register(ModuleNetwork, 15008, "failed to read the response")
	ErrFailuresResponsed          = errors.Register(ModuleNetwork, 15009, "received failed response")
)

func Wrap(err0 error, err1 error) error {
	module, code, _ := errors.ABCIInfo(err0, false)
	if err1 == nil {
		return errors.Wrapf(err0, "%s error: code: Code(%d) desc", module, code)
	} else {
		return errors.Wrapf(err0, "%s: %s error: code: (%d) desc", err1, module, code)
	}
}

func Wrapf(err error, format string, args ...interface{}) error {
	module, code, _ := errors.ABCIInfo(err, false)
	info := fmt.Sprintf(": %s error: code: Code(%d) desc", module, code)
	return errors.Wrapf(err, format+info, args...)
}
