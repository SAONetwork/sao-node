package types

// TODO: optimizae: OrderStats and OrderShards use comma split string

import (
	"strconv"
	"strings"

	saotypes "github.com/SaoNetwork/sao/x/sao/types"

	"github.com/ipfs/go-cid"
)

type CommitHeader struct {
	Controllers []string
	Labels      []string
	Schema      string
	DataType    string // datamodel, file, record
}

type DataCommit struct {
	Content any
}

type GenesisCommit struct {
	Header  CommitHeader
	Content any
}

type RawCommit struct {
	Content any
	Header  CommitHeader
	Id      cid.Cid
	Prev    cid.Cid
}

type OrderMeta struct {
	Owner                 string
	GroupId               string
	Alias                 string
	Tags                  []string
	Duration              int32
	Replica               int32
	CompleteTimeoutBlocks int
	Cid                   cid.Cid
	Rule                  string
	ExtendInfo            string
	IsUpdate              bool

	DataId    string
	OrderId   uint64
	CommitId  string
	ChunkCids []string
	TxId      string
	TxSent    bool
	Version   string
}

type MetadataProposal struct {
	Proposal     saotypes.QueryProposal
	JwsSignature saotypes.JwsSignature
}

type MetadataProposalCbor struct {
	Proposal     QueryProposal
	JwsSignature JwsSignature
}

type JwsSignature struct {
	Protected string
	Signature string
}
type QueryProposal struct {
	Owner           string
	Keyword         string
	GroupId         string
	KeywordType     uint64
	LastValidHeight uint64
	Gateway         string
	CommitId        string
	Version         string
}

type PermissionProposal struct {
	Proposal     saotypes.PermissionProposal
	JwsSignature saotypes.JwsSignature
}

type OrderStoreProposal struct {
	Proposal     saotypes.Proposal
	JwsSignature saotypes.JwsSignature
}

type OrderRenewProposal struct {
	Proposal     saotypes.RenewProposal
	JwsSignature saotypes.JwsSignature
}

type OrderTerminateProposal struct {
	Proposal     saotypes.TerminateProposal
	JwsSignature saotypes.JwsSignature
}

const (
	ModelTypes = "adsf"
)

type ModelType string

const (
	ModelTypeData = ModelType("DATA")
	ModelTypeFile = ModelType("FILE")
	ModelTypeRule = ModelType("RULE")
)

type ConsensusProposal interface {
	Marshal() ([]byte, error)
}

type MetaCommit struct {
	CommitId string
	Height   uint64
}

func ParseMetaCommit(mc string) (MetaCommit, error) {
	s := strings.Split(mc, "\032")
	if len(s) != 2 {
		return MetaCommit{}, Wrapf(ErrInvalidCommitInfo, "invalid metadata commit: %s", mc)
	}
	// TODO: validate commit id format.
	height, err := strconv.ParseUint(s[1], 10, 64)
	if err != nil {
		return MetaCommit{}, Wrapf(ErrInvalidCommitInfo, "can't parse height in metadata commit: %s: %v", mc, err)
	}
	return MetaCommit{
		CommitId: s[0],
		Height:   height,
	}, nil
}
