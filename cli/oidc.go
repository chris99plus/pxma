package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zitadel/oidc/v2/pkg/client/rp/cli"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
)

var (
	ISSUER        = "https://idp-test.htwg-konstanz.de"
	CLIENT_ID     = "https://web.pxma.christians-software-schmiede.de/"
	CLIENT_SECRET = "GrZUMNmPxTIV0BzEI1diYbTz7"
	SCOPES        = "openid profile email offline_access"

	CALLBACK_PATH = "/auth/callback"
	COOKIE_KEY    = []byte("test1234test1234")
)

var (
	ErrOIDCInterrupt = errors.New("OIDC authentication interrupted")
)

type OIDCResponse struct {
	AccessToken  string
	RefreshToken string
	IdToken      string

	Subject string

	Name      string
	GivenName string
	Email     string
}

// OIDC handler is based github.com/zitadel/oidc examples
// and based on https://github.com/openziti-test-kitchen/zssh/blob/main/zsshlib/oidc.go
func OIDCAuthenticate(ctx context.Context, lPort int) (OIDCResponse, error) {
	resCh := make(chan OIDCResponse, 1)
	errCh := make(chan error, 1)

	redirectURI := fmt.Sprintf("http://localhost:%d%v", lPort, CALLBACK_PATH)
	cookieHandler := httphelper.NewCookieHandler(COOKIE_KEY, COOKIE_KEY, httphelper.WithUnsecure())

	client := &http.Client{
		Timeout: time.Minute,
	}

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
		rp.WithHTTPClient(client),

		rp.WithPKCE(cookieHandler),
	}

	provider, err := rp.NewRelyingPartyOIDC(ctx, ISSUER, CLIENT_ID, CLIENT_SECRET, redirectURI, strings.Split(SCOPES, " "), options...)
	if err != nil {
		return OIDCResponse{}, err
	}

	// generate some state (representing the state of the user in your application,
	// e.g. the page where he was before sending him to login
	state := func() string {
		return uuid.New().String()
	}

	urlOptions := []rp.URLParamOpt{
		//rp.WithPromptURLParam("Welcome back!"),
	}

	// register the AuthURLHandler at your preferred path.
	// the AuthURLHandler creates the auth request and redirects the user to the auth server.
	// including state handling with secure cookie and the possibility to use PKCE.
	// Prompts can optionally be set to inform the server of
	// any messages that need to be prompted back to the user.
	http.Handle("/login", rp.AuthURLHandler(
		state,
		provider,
		urlOptions...,
	))

	// for demonstration purposes the returned userinfo response is written as JSON object onto response
	marshalUserinfo := func(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string, rp rp.RelyingParty, info *oidc.UserInfo) {
		resCh <- OIDCResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			IdToken:      tokens.IDToken,

			Subject: info.Subject,

			Name:      info.Name,
			GivenName: info.GivenName,
			Email:     info.Email,
		}

		msg := "<script type=\"text/javascript\">window.close()</script><body onload=\"window.close()\">You may close this window</body><p><strong>Success!</strong></p>"
		msg = msg + fmt.Sprintf("<p>Welcome %s.</p>", info.Name)
		w.Write([]byte(msg))
	}

	http.Handle(CALLBACK_PATH, rp.CodeExchangeHandler(rp.UserinfoCallback(marshalUserinfo), provider))

	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", lPort)}

	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			errCh <- err
			return
		}
	}()

	loginUrl := fmt.Sprintf("http://localhost:%d/login", lPort)
	cli.OpenBrowser(loginUrl)
	fmt.Printf("Authenticate yourself inside the Browser. If the Browser does not open naviagate to %s\n", loginUrl)

	res := OIDCResponse{}

	select {
	case <-ctx.Done():
		err = ErrOIDCInterrupt
	case r := <-resCh:
		res = r
	case e := <-errCh:
		err = e
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, 5*time.Second)
	defer cancelShutdown()
	server.Shutdown(shutdownCtx)

	return res, err
}
