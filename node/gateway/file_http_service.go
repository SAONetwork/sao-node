package gateway

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mitchellh/go-homedir"

	"sao-node/node/config"
)

var secret = []byte("SAO Network")

type HttpFileServer struct {
	Cfg    *config.SaoHttpFileServer
	Server *echo.Echo
}

type jwtClaims struct {
	Key string `json:"key"`
	jwt.StandardClaims
}

func StartHttpFileServer(cfg *config.SaoHttpFileServer) (*HttpFileServer, error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	if cfg.EnableHttpFileServerLog {
		// Middleware
		e.Use(middleware.Logger())
		e.Use(middleware.Recover())
	}

	// Unauthenticated entry
	e.GET("/test", test)

	path, err := homedir.Expand(cfg.HttpFileServerPath)
	if err != nil {
		return nil, err
	}

	handler := http.FileServer(http.Dir(path))

	// Configure middleware with the custom claims type
	config := middleware.JWTConfig{
		Claims:     &jwtClaims{},
		SigningKey: secret,
	}
	e.GET("/saonetwork/*", echo.WrapHandler(http.StripPrefix("/saonetwork/", handler)), middleware.JWTWithConfig(config))

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

	return &HttpFileServer{
		Cfg:    cfg,
		Server: e,
	}, nil
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
