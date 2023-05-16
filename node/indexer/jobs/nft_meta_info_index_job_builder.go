package jobs

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"

	"sao-node/chain"
	"sao-node/client"
	"sao-node/types"
	"sao-node/utils"

	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
)

//go:embed sqls/create_nft_meta_info_talbe.sql
var createNFTInfoDBSQL string

// {"image":"ipfs://QmRhsiTkddQ3GkVT8DZBWGRdETBSiniRSS4DaQktFHhhsu","attributes":[{"trait_type":"Background","value":"M1 Purple"},{"trait_type":"Fur","value":"M1 Red"},{"trait_type":"Eyes","value":"M1 Heart"},{"trait_type":"Hat","value":"M1 Horns"},{"trait_type":"Mouth","value":"M1 Bored"}]}
type NFTMetaInfo struct {
	Image      string
	Attributes []Attributes
}

type Attributes struct {
	TraitType string
	Value     string
}

func BuildNFTMetaInfoIndexJob(ctx context.Context, chainSvc *chain.ChainSvc, db *sql.DB, dataIds []string) *types.Job {
	// initialize the sp shard database tables
	log.Info("creating sp shard tables...")
	if _, err := db.ExecContext(ctx, createSpShardDBSQL); err != nil {
		log.Error("failed to create tables: ", err)
	}
	log.Info("creating sp shard tables done.")

	keyName := "alice"
	keyringHome := "./sao-cli/keyring"
	groupId := "GroupId"
	gatewayAddress := "saoxxxxxxxx"

	execFn := func(ctx context.Context, _ []interface{}) (interface{}, error) {
		opt := client.SaoClientOptions{
			Repo:        "./sao-cli",
			Gateway:     "http://127.0.0.1:5151",
			ChainAddr:   "http://127.0.0.1:1317",
			KeyName:     keyName,
			KeyringHome: "./sao-cli/keyring",
		}
		cli, closer, err := client.NewSaoClient(ctx, opt)
		if err != nil {
			return nil, err
		}
		defer closer()

		address, err := chain.GetAddress(ctx, keyringHome, keyName)
		if err != nil {
			return nil, err
		}

		payload := fmt.Sprintf("cosmos %s allows to generate did", address)
		secret, err := chain.SignByAccount(ctx, keyringHome, keyName, []byte(payload))
		if err != nil {
			return nil, types.Wrap(types.ErrSignedFailed, err)
		}

		provider, err := saokey.NewSecp256k1Provider(secret)
		if err != nil {
			return nil, types.Wrap(types.ErrCreateProviderFailed, err)
		}
		resolver := saokey.NewKeyResolver()

		didManager := saodid.NewDidManager(provider, resolver)
		_, err = didManager.Authenticate([]string{}, "")
		if err != nil {
			return nil, types.Wrap(types.ErrAuthenticateFailed, err)
		}

		for _, dataId := range dataIds {
			proposal := saotypes.QueryProposal{
				Owner:       didManager.Id,
				Keyword:     dataId,
				GroupId:     groupId,
				KeywordType: 2,
			}

			lastHeight, err := chainSvc.GetLastHeight(ctx)
			if err != nil {
				return nil, types.Wrap(types.ErrQueryHeightFailed, err)
			}

			peerInfo, err := chainSvc.GetNodePeer(ctx, gatewayAddress)
			if err != nil {
				return nil, err
			}

			proposal.LastValidHeight = uint64(lastHeight + 200)
			proposal.Gateway = peerInfo

			proposalBytes, err := proposal.Marshal()
			if err != nil {
				return nil, types.Wrap(types.ErrMarshalFailed, err)
			}

			jws, err := didManager.CreateJWS(proposalBytes)
			if err != nil {
				return nil, types.Wrap(types.ErrCreateJwsFailed, err)
			}

			request := &types.MetadataProposal{
				Proposal: proposal,
				JwsSignature: saotypes.JwsSignature{
					Protected: jws.Signatures[0].Protected,
					Signature: jws.Signatures[0].Signature,
				},
			}

			nftMetaInfo, err := cli.ModelLoad(ctx, request)
			if err != nil {
				return nil, err
			}

			var metaInfo NFTMetaInfo
			err = json.Unmarshal([]byte(nftMetaInfo.Content), &metaInfo)
			if err != nil {
				log.Error(err)
				return nil, err
			}

			for _, attribute := range metaInfo.Attributes {
				stmt := fmt.Sprintf("INSERT INTO NFT_trait (NFT_ID, trait_type, value) VALUES %s %s %s ",
					dataId, attribute.TraitType, attribute.Value)
				_, err := db.Exec(stmt)
				if err != nil {
					return nil, err
				}
				log.Infof("done")
			}
		}

		return nil, nil
	}

	return &types.Job{
		ID:          utils.GenerateDataId("job-id"),
		Description: "build sp shard index for order with specified sp address",
		Status:      types.JobStatusPending,
		ExecFunc:    execFn,
		Args:        make([]interface{}, 0),
	}
}
