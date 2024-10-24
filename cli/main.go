package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

var (
	ErrInvalidOptions = errors.New("invalid options")
)

func main() {
	var identityJson string
	var ctrlUrl string
	var useOIDC bool

	flag.StringVar(&identityJson, "id", "", "Path to the identity json file")
	flag.StringVar(&ctrlUrl, "ctrl", "", "Url of the ziti controller")
	flag.BoolVar(&useOIDC, "oidc", false, "Enable and use the HTWG OIDC provider")

	flag.Parse()

	o := NewOptions()

	if identityJson != "" {
		o.WithIdentityJson(identityJson)
	}

	if useOIDC {
		o.WithOIDC()
	}

	if ctrlUrl != "" {
		o.WithController(ctrlUrl)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	err := EmailProxy(ctx, o)
	if err == ErrInvalidOptions {
		fmt.Printf("Usage: HTWG Konstanz Email Proxy\n")
		fmt.Printf("%s [-id <path>] [-ctrl <url>] [-oidc]\n", os.Args[0])
		os.Exit(2)
	} else if err != nil {
		panic(err)
	}

	os.Exit(0)
}
