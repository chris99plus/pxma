package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/openziti/edge-api/rest_model"
	edgeapis "github.com/openziti/sdk-golang/edge-apis"
	"github.com/openziti/sdk-golang/ziti"
)

const (
	SMTP_LISTEN_PORT  = 25
	SMTP_SERVICE_NAME = "SMTPEmail"

	IMAP_LISTEN_PORT  = 143
	IMAP_SERVICE_NAME = "IMAPEmail"
)

func ListenSMTP(ctx context.Context, zitiCtx ziti.Context) {
	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", SMTP_LISTEN_PORT))
	if err != nil {
		panic(err)
	}

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	fmt.Printf("Waiting on %s for SMTP traffic\n", l.Addr().String())

	for {
		cConn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		go func() {
			listenCtx, listenCancel := context.WithCancel(ctx)

			fmt.Printf("-- Accepted connection from %s for SMTP\n", cConn.RemoteAddr().String())
			defer cConn.Close()

			sConn, err := zitiCtx.Dial(SMTP_SERVICE_NAME)
			if err != nil {
				panic(err)
			}
			defer sConn.Close()

			// Copy data from service to local
			go func() {
				_, err := io.Copy(sConn, cConn)
				if err != nil {
					fmt.Printf("-- %s: Service -> Local: Copy error: %s\n", cConn.RemoteAddr().String(), err)
				}

				listenCancel()
			}()

			// Copy data from local to service
			go func() {
				_, err := io.Copy(cConn, sConn)
				if err != nil {
					fmt.Printf("-- %s: Local -> Service: Copy error: %s\n", cConn.RemoteAddr().String(), err)
				}

				listenCancel()
			}()

			<-listenCtx.Done()
		}()
	}
}

func ListenIMAP(ctx context.Context, zitiCtx ziti.Context) {
	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", IMAP_LISTEN_PORT))
	if err != nil {
		panic(err)
	}

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	fmt.Printf("Waiting on %s for IMAP traffic\n", l.Addr().String())

	for {
		cConn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		go func() {
			listenCtx, listenCancel := context.WithCancel(ctx)

			fmt.Printf("-- Accepted connection from %s for IMAP\n", cConn.RemoteAddr().String())
			defer cConn.Close()

			sConn, err := zitiCtx.Dial(IMAP_SERVICE_NAME)
			if err != nil {
				panic(err)
			}
			defer sConn.Close()

			// Copy data from service to local
			go func() {
				_, err := io.Copy(sConn, cConn)
				if err != nil {
					fmt.Printf("-- %s: Service -> Local: Copy error: %s\n", cConn.RemoteAddr().String(), err)
				}

				listenCancel()
			}()

			// Copy data from local to service
			go func() {
				_, err := io.Copy(cConn, sConn)
				if err != nil {
					fmt.Printf("-- %s: Local -> Service: Copy error: %s\n", cConn.RemoteAddr().String(), err)
				}

				listenCancel()
			}()

			<-listenCtx.Done()
		}()
	}
}

func RegisterEvents(zitiCtx ziti.Context) {
	zitiCtx.Events().AddMfaTotpCodeListener(func(zitiCtx ziti.Context, aqd *rest_model.AuthQueryDetail, mcr ziti.MfaCodeResponse) {
		if aqd.HTTPURL != "http://localhost:8080/login" {
			fmt.Printf("MFA Required. Please enter your code: ")
			attempt := 1
			for {

				reader := bufio.NewReader(os.Stdin)
				text, _ := reader.ReadString('\n')
				code := strings.Trim(text, "\n")

				err := mcr(code)
				if err != nil && attempt >= 3 {
					fmt.Println("ERROR. To many retries")
					os.Exit(1)
				} else if err != nil {
					attempt += 1
					fmt.Printf("Invalid Code. Try again: ")
				} else {
					return
				}
			}
		}
	})

	zitiCtx.Events().AddRouterConnectedListener(func(ztx ziti.Context, name, addr string) {
		fmt.Printf("--- ROUTER %s CONNECTED: %s\n", name, addr)
	})

	zitiCtx.Events().AddRouterDisconnectedListener(func(ztx ziti.Context, name, addr string) {
		fmt.Printf("--- ROUTER %s DISCONNECTED: %s\n", name, addr)
	})

	zitiCtx.Events().AddAuthenticationStateUnauthenticatedListener(func(ctx ziti.Context, as edgeapis.ApiSession) {
		fmt.Printf("--- Session unauthenticated: %s\n", as.GetIdentityName())
	})

	zitiCtx.Events().AddAuthenticationStatePartialListener(func(ctx ziti.Context, as edgeapis.ApiSession) {
		fmt.Printf("--- Session partial authenticated: %s\n", as.GetIdentityName())
	})

	zitiCtx.Events().AddAuthenticationStateFullListener(func(ctx ziti.Context, as edgeapis.ApiSession) {
		fmt.Printf("--- Session authenticated: %s\n", as.GetIdentityName())
	})
}

// func enrollMfa(client *edge_apis.ClientApiClient, session edge_apis.ApiSession, deleteMfa bool) {
// 	if deleteMfa {
// 		_, err := client.API.CurrentIdentity.DeleteMfa(current_identity.NewDeleteMfaParams(), session)
// 		if err != nil {
// 			fmt.Printf("ERROR deleting mfa: %s\n", err)
// 		}
// 	}
//
// 	mfa_create, err := client.API.CurrentIdentity.EnrollMfa(current_identity.NewEnrollMfaParams(), session)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(mfa_create.Payload.Data)
//
// 	mfa_detail, err := client.API.CurrentIdentity.DetailMfa(current_identity.NewDetailMfaParams(), session)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(mfa_detail.Payload.Data)
//
// 	reader := bufio.NewReader(os.Stdin)
// 	fmt.Print("Enter code: ")
// 	text, _ := reader.ReadString('\n')
// 	code := strings.Trim(text, "\n")
// 	fmt.Printf("Your entered: \"%s\"\n", code)
//
// 	verify, err := client.API.CurrentIdentity.VerifyMfa(current_identity.NewVerifyMfaParams().WithMfaValidation(&rest_model.MfaCode{Code: &code}), session)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Println(verify.Payload.Data)
// }
