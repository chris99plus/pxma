package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/openziti/edge-api/rest_client_api_client/current_identity"
	"github.com/openziti/edge-api/rest_model"
	edge_apis "github.com/openziti/sdk-golang/edge-apis"
	"github.com/openziti/sdk-golang/ziti"
)

func main() {

	cfg, err := ziti.NewConfigFromFile("/home/chris/Downloads/testssh.json")
	if err != nil {
		panic(fmt.Sprintf("Config error: %s", err))
	}

	// apiUrl, _ := url.Parse("https://ctrl.pxma.christians-software-schmiede.de:1280/edge/client/v1")
	// cred := edge_apis.NewIdentityCredentialsFromConfig(cfg.ID)
	// totpCallback := func(s chan string) {
	// fmt.Println("Require totp")
	// someval := <-s
	// fmt.Println(someval)
	// }

	// client := edge_apis.NewClientApiClient([]*url.URL{apiUrl}, cred.GetCaPool(), totpCallback)

	// _, err = client.Authenticate(cred, []string{})
	// if err != nil {
	// panic(fmt.Sprintf("Authenticate error: %s", err))
	// }

	// enrollMfa(client, session, false)

	ctx, err := ziti.NewContext(cfg) //get a ziti context using a file
	if err != nil {
		panic(err)
	}

	ctx.Events().AddMfaTotpCodeListener(func(ctx ziti.Context, aqd *rest_model.AuthQueryDetail, mcr ziti.MfaCodeResponse) {
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
	})

	ctx.Events().AddRouterConnectedListener(func(ztx ziti.Context, name, addr string) {
		fmt.Printf("ROUTER %s CONNECTED: %s\n", name, addr)
	})

	err = ctx.Authenticate()
	if err != nil {
		panic(err)
	}

	foundSvc, ok := ctx.GetService("SMTPEmail")
	fmt.Printf("%s, %t\n", *foundSvc.ID, ok)

	for _, el := range foundSvc.PostureQueries {
		fmt.Printf("Posture Queries\n\tIsPassing: %t\n", *el.IsPassing)
		fmt.Printf("\tPolicy Type: %s, Id: %s\n", el.PolicyType, *el.PolicyID)
	}
	fmt.Println(foundSvc.PostureQueries)

	go func() {

		clientLn, err := net.Listen("tcp", ":1443")
		if err != nil {
			panic(err)
		}

		fmt.Printf("Waiting on %s for a TCP connection to be established\n", clientLn.Addr().String())

		for {
			clientConn, err := clientLn.Accept()
			if err != nil {
				panic(err)
			}

			fmt.Printf("Accepted connection from %s\n", clientConn.RemoteAddr().String())

			fmt.Println("Dialing IMAP Email service")
			serviceConn, err := ctx.Dial("IMAPEmail")
			// serviceConn, err := ctx.DialWithOptions("SMTP Email", dialOptions)

			// serviceConn, err := ctx.Dial("vm01ssh")
			if err != nil {
				panic(err)
			}

			go io.Copy(serviceConn, clientConn)
			io.Copy(clientConn, serviceConn)

			// serviceConn.Close()
			// clientConn.Close()
		}
	}()
	// dialOptions := &ziti.DialOptions{
	// 	ConnectTimeout: 0,
	// 	Identity:       "vm202.pxma Router",
	// 	AppData:        nil,
	// }

	// serviceConn, err := ctx.DialWithOptions("SMTP Email", dialOptions)

	// serviceConn, err := ctx.Dial("vm01ssh")

	clientLn, err := net.Listen("tcp", ":2525")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Waiting on %s for a TCP connection to be established\n", clientLn.Addr().String())

	for {
		clientConn, err := clientLn.Accept()
		if err != nil {
			panic(err)
		}

		fmt.Printf("Accepted connection from %s\n", clientConn.RemoteAddr().String())

		fmt.Println("Dialing SMTP Email service")

		serviceConn, err := ctx.Dial("SMTPEmail")
		if err != nil {
			panic(err)
		}

		go func() {
			_, err := io.Copy(serviceConn, clientConn)
			if err != nil {
				fmt.Printf("SMTP copy error: %s\n", err)
			}
		}()
		_, err = io.Copy(clientConn, serviceConn)
		if err != nil {
			fmt.Printf("SMTP copy error: %s\n", err)
		}

		// serviceConn.Close()
		// clientConn.Close()
	}
	// mfa, err := client.API.CurrentIdentity.DetailMfa(current_identity.NewDetailMfaParams(), session)
	// if err != nil {
	// panic(err)
	// }
	// fmt.Printf("MFA: %s\n", mfa.Payload.Data)
}

func enrollMfa(client *edge_apis.ClientApiClient, session edge_apis.ApiSession, deleteMfa bool) {
	if deleteMfa {
		_, err := client.API.CurrentIdentity.DeleteMfa(current_identity.NewDeleteMfaParams(), session)
		if err != nil {
			fmt.Printf("ERROR deleting mfa: %s\n", err)
		}
	}

	mfa_create, err := client.API.CurrentIdentity.EnrollMfa(current_identity.NewEnrollMfaParams(), session)
	if err != nil {
		panic(err)
	}

	fmt.Println(mfa_create.Payload.Data)

	mfa_detail, err := client.API.CurrentIdentity.DetailMfa(current_identity.NewDetailMfaParams(), session)
	if err != nil {
		panic(err)
	}

	fmt.Println(mfa_detail.Payload.Data)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter code: ")
	text, _ := reader.ReadString('\n')
	code := strings.Trim(text, "\n")
	fmt.Printf("Your entered: \"%s\"\n", code)

	verify, err := client.API.CurrentIdentity.VerifyMfa(current_identity.NewVerifyMfaParams().WithMfaValidation(&rest_model.MfaCode{Code: &code}), session)
	if err != nil {
		panic(err)
	}

	fmt.Println(verify.Payload.Data)
}
