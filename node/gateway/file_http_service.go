package gateway

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	did "github.com/SaoNetwork/sao-did"
	saotypes "github.com/SaoNetwork/sao/x/sao/types"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	Cfg        *config.SaoHttpFileServer
	NodeCFG    *config.Node
	Server     *echo.Echo
	cctx       *cli.Context
	ServerPath string
}

type jwtClaims struct {
	Key string `json:"key"`
	jwt.StandardClaims
}

func StartHttpFileServer(serverPath string, cfg *config.SaoHttpFileServer, ncfg *config.Node, cctx *cli.Context) (*HttpFileServer, error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	log.Infof("start http server server path: %s", serverPath)

	if cfg.EnableHttpFileServerLog {
		// Middleware
		e.Use(middleware.Logger())
		e.Use(middleware.Recover())
	}

	// Unauthenticated entry
	e.GET("/test", test)

	/*
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
	*/

	s := &HttpFileServer{
		Cfg:        cfg,
		NodeCFG:    ncfg,
		Server:     e,
		cctx:       cctx,
		ServerPath: serverPath,
	}
	e.GET("/sao/*", s.load)

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

	log.Info("proposalbyte", string(proposalBytes))

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
		return err
	}

	defer closer()

	req := ec.Request()
	uri := strings.TrimLeft(req.URL.String(), "/sao/")
	log.Info(uri)

	re, _ := regexp.Compile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$")
	uuid := re.FindString(uri)
	var keyword string
	var groupId string
	if uuid != "" {
		keyword = uuid
	} else {
		params := strings.SplitN(uri, "/", 3)
		if len(params) == 3 {
			key := fmt.Sprintf("%s-%s-%s", params[0], params[2], params[1])
			model, err := c.GetModel(ctx, key)
			log.Info(model, err, key)
			if err == nil {
				keyword = model.Model.Data
			}
		}
	}

	if keyword == "" {
		ec.String(http.StatusNotFound, "invalid request")
		return nil
	}

	didManager, _, err := GetDidManager(ctx, c.Cfg.KeyName)
	if err != nil {
		return err
	}

	proposal := saotypes.QueryProposal{
		Owner:       didManager.Id,
		Keyword:     keyword,
		GroupId:     groupId,
		KeywordType: 1,
	}

	if !utils.IsDataId(keyword) {
		proposal.KeywordType = 2
	}

	gatewayAddress, err := c.GetNodeAddress(ctx)
	if err != nil {
		return err
	}

	request, err := buildQueryRequest(ctx, didManager, proposal, c, gatewayAddress)
	if err != nil {
		return err
	}

	resp, err := c.ModelLoad(ctx, request)
	if err != nil {
		if strings.Index(err.Error(), "NotFound") > 0 {
			ec.String(http.StatusNotFound, "model not found")
			return nil
		}
		return err
	}

	cacheFile := path.Join(h.ServerPath, resp.DataId)

	return ec.File(cacheFile)
}
