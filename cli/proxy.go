package main

import (
	"context"
	"fmt"
	"strings"

	edgeapis "github.com/openziti/sdk-golang/edge-apis"
	"github.com/openziti/sdk-golang/ziti"
)

type EmailProxyOptions struct {
	IdentityJson    string
	useIdentityFile bool

	ControllerUrl string
	OIDCEnabled   bool
	useOIDC       bool
}

func (e *EmailProxyOptions) defaults() error {
	if e.IdentityJson != "" {
		e.useIdentityFile = true
	}

	if e.OIDCEnabled && (e.ControllerUrl != "" || e.IdentityJson != "") {
		e.useOIDC = true
	}

	if !e.useIdentityFile && !e.useOIDC {
		return ErrInvalidOptions
	}

	return nil
}

func (e *EmailProxyOptions) WithOIDC() EmailProxyOptions {
	e.useOIDC = true
	return *e
}

func (e *EmailProxyOptions) WithController(ctrlUrl string) EmailProxyOptions {
	e.ControllerUrl = ctrlUrl
	return *e
}

func (e *EmailProxyOptions) WithIdentityJson(file string) EmailProxyOptions {
	e.IdentityJson = file
	return *e
}

func NewOptions() EmailProxyOptions {
	return EmailProxyOptions{}
}

func EmailProxy(ctx context.Context, options EmailProxyOptions) error {
	if err := options.defaults(); err != nil {
		return err
	}

	var cfg *ziti.Config
	var oidcRes OIDCResponse
	var err error

	if options.useIdentityFile {
		cfg, err = ziti.NewConfigFromFile(options.IdentityJson)
		if err != nil {
			return err
		}
	}

	if options.useOIDC {
		oidcRes, err = OIDCAuthenticate(ctx, 8080)
		if err != nil && err != ErrOIDCInterrupt {
			return err
		} else if err != nil {
			fmt.Println("Authentication interrupted ...")
			return nil
		}
	}

	if options.useOIDC && !options.useIdentityFile {
		cfg, err = NewZitiJwtConfig(options.ControllerUrl, oidcRes.IdToken)
		if err != nil {
			return err
		}
	}

	zitiCtx, err := ziti.NewContext(cfg)
	if err != nil {
		panic(err)
	}
	defer zitiCtx.Close()

	if options.useIdentityFile && options.useOIDC {
		cfg.Credentials.AddJWT(oidcRes.IdToken)
	}

	RegisterEvents(zitiCtx)

	err = zitiCtx.Authenticate()
	if err != nil {
		panic(err)
	}

	go ListenIMAP(ctx, zitiCtx)
	go ListenSMTP(ctx, zitiCtx)

	<-ctx.Done()

	return nil
}

func NewZitiJwtConfig(ctrlUrl string, oidcToken string) (*ziti.Config, error) {
	if !strings.Contains(ctrlUrl, "://") {
		ctrlUrl = "https://" + ctrlUrl
	}

	caPool, err := ziti.GetControllerWellKnownCaPool(ctrlUrl)
	if err != nil {
		return nil, err
	}

	credentials := edgeapis.NewJwtCredentials(oidcToken)
	credentials.CaPool = caPool

	cfg := &ziti.Config{
		ZtAPI:       ctrlUrl + "/edge/client/v1",
		Credentials: credentials,
	}

	credentials.AddJWT(oidcToken) // satisfy the ext-jwt-auth primary + secondary
	cfg.ConfigTypes = append(cfg.ConfigTypes, "all")

	return cfg, nil
}
