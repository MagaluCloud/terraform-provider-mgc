package main

import (
	"context"
	"flag"
	"log"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var Version string = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/magalucloud/mgc",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), mgc.New(Version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
