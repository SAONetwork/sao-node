package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	did "github.com/SaoNetwork/sao-did"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"sao-node/chain"
	"sao-node/client"
	"sao-node/node/config"
	"sao-node/types"
	"sao-node/utils"

	saodid "github.com/SaoNetwork/sao-did"
	saokey "github.com/SaoNetwork/sao-did/key"
)

const (
	FlagClientRepo = "repo"
	FlagKeyName    = "key-name"
)

var secret = []byte("SAO Network")

type HttpFileServer struct {
	Cfg     *config.SaoHttpFileServer
	NodeCFG *config.Node
	Server  *echo.Echo
	cctx    *cli.Context
}

type jwtClaims struct {
	Key string `json:"key"`
	jwt.StandardClaims
}

func StartHttpFileServer(serverPath string, cfg *config.SaoHttpFileServer, ncfg *config.Node, cctx *cli.Context) (*HttpFileServer, error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	log.Info("start http server")
	fmt.Println("start http server")

	if cfg.EnableHttpFileServerLog {
		// Middleware
		e.Use(middleware.Logger())
		e.Use(middleware.Recover())
	}

	// Unauthenticated entry
	e.GET("/test", test)

	path, err := homedir.Expand(serverPath)
	if err != nil {
		return nil, types.Wrap(types.ErrInvalidPath, err)
	}

	handler := http.FileServer(http.Dir(path))

	// Configure middleware with the custom claims type
	config := middleware.JWTConfig{
		Claims:     &jwtClaims{},
		SigningKey: secret,
	}

	s := &HttpFileServer{
		Cfg:     cfg,
		NodeCFG: ncfg,
		Server:  e,
		cctx:    cctx,
	}
	e.GET("/saonetwork/*", echo.WrapHandler(http.StripPrefix("/saonetwork/", handler)), middleware.JWTWithConfig(config))
	//e.GET("/v1/^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$", s.load)
	e.GET("/v1/*", s.load)

	go func() {
		err := e.Start(cfg.HttpFileServerAddress)
		if err != nil {
			if strings.Contains(err.Error(), "Server closed") {
				log.Info("stopping file http service...")
			} else {
				log.Error(err.Error())
			}
		}
	}()
	return s, nil
}

func (hfs *HttpFileServer) Stop(ctx context.Context) error {
	return hfs.Server.Shutdown(ctx)
}

func (hfs *HttpFileServer) GenerateToken(owner string) (string, string) {
	// Set custom claims
	claims := &jwtClaims{
		owner,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(hfs.Cfg.TokenPeriod).Unix(),
		},
	}

	// ModelCreate token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if token == nil {
		log.Error("failed to generate token")
		return "", ""
	}

	// Generate encoded token and send it as response.
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		log.Error(err.Error())
		return "", ""
	}

	return hfs.Cfg.HttpFileServerAddress, tokenStr
}

func test(c echo.Context) error {
	return c.String(http.StatusOK, "Accessible")
}

func GetDidManager(ctx context.Context, keyName string) (*saodid.DidManager, string, error) {

	KeyringHome := "~/.sao"

	address, err := chain.GetAddress(ctx, KeyringHome, keyName)
	if err != nil {
		return nil, "", err
	}

	payload := fmt.Sprintf("cosmos %s allows to generate did", address)
	secret, err := chain.SignByAccount(ctx, KeyringHome, keyName, []byte(payload))
	if err != nil {
		return nil, "", types.Wrap(types.ErrSignedFailed, err)
	}

	provider, err := saokey.NewSecp256k1Provider(secret)
	if err != nil {
		return nil, "", types.Wrap(types.ErrCreateProviderFailed, err)
	}
	resolver := saokey.NewKeyResolver()

	didManager := saodid.NewDidManager(provider, resolver)
	_, err = didManager.Authenticate([]string{}, "")
	if err != nil {
		return nil, "", types.Wrap(types.ErrAuthenticateFailed, err)
	}

	return &didManager, address, nil
}

func buildQueryRequest(ctx context.Context, didManager *did.DidManager, proposal saotypes.QueryProposal, chain chain.ChainSvcApi, gatewayAddress string) (*types.MetadataProposal, error) {
	lastHeight, err := chain.GetLastHeight(ctx)
	if err != nil {
		return nil, types.Wrap(types.ErrQueryHeightFailed, err)
	}

	peerInfo, err := chain.GetNodePeer(ctx, gatewayAddress)
	if err != nil {
		return nil, err
	}

	proposal.LastValidHeight = uint64(lastHeight + 200)
	proposal.Gateway = peerInfo

	if proposal.Owner == "all" {
		return &types.MetadataProposal{
			Proposal: proposal,
		}, nil
	}

	proposalBytes, err := proposal.Marshal()
	if err != nil {
		return nil, types.Wrap(types.ErrMarshalFailed, err)
	}

	jws, err := didManager.CreateJWS(proposalBytes)
	if err != nil {
		return nil, types.Wrap(types.ErrCreateJwsFailed, err)
	}

	return &types.MetadataProposal{
		Proposal: proposal,
		JwsSignature: saotypes.JwsSignature{
			Protected: jws.Signatures[0].Protected,
			Signature: jws.Signatures[0].Signature,
		},
	}, nil
}

func (h *HttpFileServer) load(ec echo.Context) error {

	fmt.Println("aaaaaaa")

	keyringHome := "~/.sao"
	keyName := "client"
	opt := client.SaoClientOptions{
		Repo:        "~/.sao-cli",
		Gateway:     "http://127.0.0.1:5151/rpc/v0",
		ChainAddr:   h.NodeCFG.Chain.Remote,
		KeyName:     keyName,
		KeyringHome: keyringHome,
	}

	ctx := context.Background()
	c, closer, err := client.NewSaoClient(ctx, opt)
	if err != nil {
		fmt.Println("1")
		return err
	}

	defer closer()

	didManager, _, err := GetDidManager(ctx, c.Cfg.KeyName)
	if err != nil {
		fmt.Println("2")
		return err
	}

	keyword := "6f0041a4-efdf-11ed-b99a-8930faeb97d0"

	proposal := saotypes.QueryProposal{
		Owner:   didManager.Id,
		Keyword: keyword,
	}

	if !utils.IsDataId(keyword) {
		proposal.KeywordType = 2
	}

	gatewayAddress, err := c.GetNodeAddress(ctx)
	if err != nil {
		fmt.Println("3")
		return err
	}

	request, err := buildQueryRequest(ctx, didManager, proposal, c, gatewayAddress)
	if err != nil {
		fmt.Println("4")
		return err
	}

	resp, err := c.ModelLoad(ctx, request)
	if err != nil {
		fmt.Println("5", err)
		return err
	}

	fmt.Println(resp)

	return ec.String(http.StatusOK, "Accessible")
}
