package main

import (
	"context"
	"encoding/hex"
	saoclient "sao-storage-node/client"
	"sao-storage-node/utils"
	"testing"

	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
	didtypes "github.com/SaoNetwork/sao/x/did/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/dvsekhvalnov/jose2go/base64url"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/require"

	"github.com/urfave/cli/v2"
)

var log = logging.Logger("saoclient")

var TestCmd = &cli.Command{
	Name: "test",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "abc",
			Required: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		gateway := cctx.String("gateway")

		client := saoclient.NewSaoClient(ctx, "~/.sao_cli", gateway)
		resp, err := client.Test(ctx)
		if err != nil {
			return err
		}
		log.Info(resp)
		return nil
	},
}

func TestSignature(t *testing.T) {
	ctx := context.TODO()
	gateway := ""
	client := saoclient.NewSaoClient(ctx, "~/.sao_cli", gateway)
	require.NotNil(t, client)

	secret, err := hex.DecodeString("a3709843cbd4e72d7215512e28385123b44eab5e27f36001d74ee1cff671502d")
	require.NoError(t, err)

	provider, err := saokey.NewSecp256k1Provider(secret)
	require.NoError(t, err)

	didManager := saodid.NewDidManager(provider, saokey.NewKeyResolver())
	_, err = didManager.Authenticate([]string{}, "")
	require.NoError(t, err)

	proposal := saotypes.QueryProposal{
		Owner:           didManager.Id,
		Keyword:         utils.GenerateDataId(),
		LastValidHeight: uint64(100),
		Gateway:         "peerInfo",
	}

	proposalBytes, err := proposal.Marshal()
	require.NoError(t, err)

	jws, err := didManager.CreateJWS(proposalBytes)
	require.NoError(t, err)

	didManager2, err := saodid.NewDidManagerWithDid(didManager.Id, getSidDocFunc())
	require.NoError(t, err)

	_, err = didManager2.VerifyJWS(saodidtypes.GeneralJWS{
		Payload: base64url.Encode(proposalBytes),
		Signatures: []saodidtypes.JwsSignature{
			saodidtypes.JwsSignature(jws.Signatures[0]),
		},
	})
	require.NoError(t, err)
}

func getSidDocFunc() func(versionId string) (*didtypes.SidDocument, error) {
	return func(versionId string) (*didtypes.SidDocument, error) {
		keys := make([]*didtypes.PubKey, 0)
		keys = append(keys, &didtypes.PubKey{
			Name:  "",
			Value: "",
		})
		return &didtypes.SidDocument{
			VersionId: versionId,
			Keys:      keys,
		}, nil
	}
}
