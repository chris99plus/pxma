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

	cfg, err := ziti.NewConfigFromFile("/home/chris/Downloads/CliTest.json")
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

	foundSvc, ok := ctx.GetService("vm01ssh")
	fmt.Printf("%s, %t", *foundSvc.ID, ok)

	for _, el := range foundSvc.PostureQueries {
		fmt.Printf("Posture Queries\n\tIsPassing: %t\n", *el.IsPassing)
		fmt.Printf("\tPolicy Type: %s, Id: %s\n", el.PolicyType, *el.PolicyID)
	}
	fmt.Println(foundSvc.PostureQueries)

	fmt.Println("Dialing vm01ssh service")

	dialOptions := &ziti.DialOptions{
		ConnectTimeout: 0,
		Identity:       "pxma01 Router",
		AppData:        nil,
	}
	serviceConn, err := ctx.DialWithOptions("vm01ssh", dialOptions)

	// serviceConn, err := ctx.Dial("vm01ssh")
	if err != nil {
		panic(err)
	}
	defer serviceConn.Close()

	clientLn, err := net.Listen("tcp", ":3030")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Waiting on %s for a TCP connection to be established", clientLn.Addr().String())

	clientConn, err := clientLn.Accept()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Accepted connection from %s\n", clientConn.RemoteAddr().String())

	go io.Copy(serviceConn, clientConn)
	io.Copy(clientConn, serviceConn)
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
