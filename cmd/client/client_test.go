package main

import (
	"context"
	"encoding/hex"
	saoclient "sao-node/client"
	"sao-node/utils"
	"testing"

	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	sid "github.com/SaoNetwork/sao-did/sid"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/dvsekhvalnov/jose2go/base64url"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/require"
)

var log = logging.Logger("saoclient")

func TestSignature(t *testing.T) {
	ctx := context.TODO()
	gateway := ""
	opt := saoclient.SaoClientOptions{
		Repo:      "~/.sao_cli",
		Gateway:   gateway,
		ChainAddr: "none",
	}
	client, closer, err := saoclient.NewSaoClient(ctx, opt)
	require.Nil(t, err)
	require.NotNil(t, client)
	defer closer()

	secret, err := hex.DecodeString("a3709843cbd4e72d7215512e28385123b44eab5e27f36001d74ee1cff671502d")
	require.NoError(t, err)

	provider, err := saokey.NewSecp256k1Provider(secret)
	require.NoError(t, err)

	didManager := saodid.NewDidManager(provider, saokey.NewKeyResolver())
	_, err = didManager.Authenticate([]string{}, "")
	require.NoError(t, err)

	proposal := saotypes.QueryProposal{
		Owner:           didManager.Id,
		Keyword:         utils.GenerateDataId(didManager.Id),
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

func getSidDocFunc() func(versionId string) (*sid.SidDocument, error) {
	log.Info("getSidDocFunc...")
	return func(versionId string) (*sid.SidDocument, error) {
		keys := make([]*sid.PubKey, 0)
		keys = append(keys, &sid.PubKey{
			Name:  "",
			Value: "",
		})
		return &sid.SidDocument{
			VersionId: versionId,
			Keys:      keys,
		}, nil
	}
}
