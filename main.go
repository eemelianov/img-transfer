package main

import (
	"github.com/eemelianov/transporter/registry"
	sdk "github.com/hashicorp/waypoint-plugin-sdk"
)

func main() {
	sdk.Main(sdk.WithComponents(
		&registry.Registry{},
	))
}

//func main() {
//	var c registry.Registry
//	src := `host="ssh://eemelianov@192.168.0.47:2222"
//			image="eemelianov/plutos"
//			tag="5846a0abd47763fba11e6aa9dd5648c8e2b9524d"`
//	f, _ := hclparse.NewParser().ParseHCL([]byte(src), "test.hcl")
//
//	component.Configure(&c, f.Body, nil)
//	ui := terminal.ConsoleUI(context.Background())
//
//	pushFunc := c.PushFunc().(func(*component.Source, context.Context, terminal.UI) (*registry.Image, error))
//	pushFunc(&component.Source{App: "plutos-bot", Path: "./"}, context.Background(), ui)
//}
