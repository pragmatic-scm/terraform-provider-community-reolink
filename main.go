// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/pragmatic-scm/terraform-provider-community-reolink/internal/provider"
)

var version = "dev"

func main() {
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/pragmatic-scm/community-reolink",
	})
	if err != nil {
		log.Fatal(err)
	}
}
