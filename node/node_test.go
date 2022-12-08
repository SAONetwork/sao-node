package node

import (
	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
	saodidtypes "github.com/SaoNetwork/sao-did/types"
	"github.com/dvsekhvalnov/jose2go/arrays"
	"testing"
)

func TestKeyDidSign(t *testing.T) {
	seed, err := arrays.Random(32)
	if err != nil {
		t.Error(err)
	}
	provider, err := saokey.NewSecp256k1Provider(seed)
	if err != nil {
		t.Error(err)
	}
	resolver := saokey.NewKeyResolver()
	didManager := saodid.NewDidManager(provider, resolver)
	_, err = didManager.Authenticate([]string{}, "")
	if err != nil {
		t.Error(err)
	}

	jws, err := didManager.CreateJWS([]byte("payload"))
	if err != nil {
		t.Error(err)
	}

	_, err = didManager.VerifyJWS(saodidtypes.GeneralJWS{
		Payload:    jws.Payload,
		Signatures: jws.Signatures,
	})
	if err != nil {
		t.Errorf("verify client order proposal signature failed: %v", err)
	}

}
