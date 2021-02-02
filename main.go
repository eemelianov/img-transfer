package main

import (
	"github.com/eemelianov/img-transfer/registry"
	sdk "github.com/hashicorp/waypoint-plugin-sdk"
)

func main() {
	sdk.Main(sdk.WithComponents(
		&registry.Registry{},
	))
}
