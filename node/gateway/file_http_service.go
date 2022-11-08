package gateway

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mitchellh/go-homedir"

	"sao-storage-node/node/config"
)

var secret = []byte("SAO Network")

type HttpFileServer struct {
	Cfg    *config.Gateway
	Server *echo.Echo
}

type jwtClaims struct {
	Key string `json:"key"`
	jwt.StandardClaims
}

func StartHttpFileServer(cfg *config.Gateway) (*HttpFileServer, error) {
	e := echo.New()
	e.HideBanner = true

	if cfg.EnableHttpFileServerLog {
		// Middleware
		e.Use(middleware.Logger())
	}

	// Unauthenticated entry
	e.GET("/test", test)

	path, err := homedir.Expand(cfg.HttpFileServerPath)
	if err != nil {
		return nil, err
	}

	assetHandler := http.FileServer(http.FS(os.DirFS(path)))
	e.GET("/saonetwork/*", echo.WrapHandler(http.StripPrefix("/saonetwork/", assetHandler)))

	// Restricted entry
	r := e.Group("/saonetwork")

	// Configure middleware with the custom claims type
	config := middleware.JWTConfig{
		Claims:     &jwtClaims{},
		SigningKey: secret,
	}
	r.Use(middleware.JWTWithConfig(config))
	r.GET("", restricted)

	go func() {
		err := e.Start(cfg.HttpFileServerAddress)
		// err := e.Start(cfg.HttpFileServerAddress)
		if err != nil {
			log.Error(err.Error())
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

func (hfs *HttpFileServer) GenerateToken(owner string) string {
	// Set custom claims
	claims := &jwtClaims{
		owner,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(hfs.Cfg.TokenPeriod).Unix(),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if token == nil {
		log.Error("failed to generate token")
		return ""
	}

	// Generate encoded token and send it as response.
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	return tokenStr
}

func test(c echo.Context) error {
	return c.String(http.StatusOK, "Accessible")
}

func restricted(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*jwtClaims)
	return c.String(http.StatusOK, "got request from "+claims.Key)
}
