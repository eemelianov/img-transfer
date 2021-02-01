package main

import (
	"github.com/eemelianov/transporter/builder"
	"github.com/eemelianov/transporter/platform"
	"github.com/eemelianov/transporter/registry"
	"github.com/eemelianov/transporter/release"
	sdk "github.com/hashicorp/waypoint-plugin-sdk"
)

func main() {
	// sdk.Main allows you to register the components which should
	// be included in your plugin
	// Main sets up all the go-plugin requirements

	sdk.Main(sdk.WithComponents(
		// Comment out any components which are not
		// required for your plugin
		&builder.Builder{},
		&registry.Registry{},
		&platform.Platform{},
		&release.ReleaseManager{},
	))
}
