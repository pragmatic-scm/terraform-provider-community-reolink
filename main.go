// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"

	"github.com/eremington/terraform-provider-reolink/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/eremington/reolink",
	})
	if err != nil {
		log.Fatal(err)
	}
}
