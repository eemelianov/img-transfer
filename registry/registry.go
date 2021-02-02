package registry

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/hashicorp/waypoint-plugin-sdk/docs"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"net/http"
)

type TransporterConfig struct {
	Host  string `hcl:"host"`
	Image string `hcl:"image"`
	Tag   string `hcl:"tag"`
}

type Registry struct {
	config TransporterConfig
}

func (r *Registry) Config() (interface{}, error) {
	return &r.config, nil
}

func (r *Registry) ConfigSet(config interface{}) error {
	c, ok := config.(*TransporterConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("expected *transporterconfig as parameter")
	}

	// validate the config
	if c.Host == "" {
		return fmt.Errorf("host must be set to a valid ssh connection string")
	}
	return nil
}

func (r *Registry) PushFunc() interface{} {
	// return a function which will be called by Waypoint
	return r.push
}

func (r *Registry) push(src *Image, ctx context.Context, ui terminal.UI) (*Image, error) {
	u := ui.Status()
	defer u.Close()
	localCLI, err := localClient()
	if err != nil {
		return nil, err
	}
	defer localCLI.Close()
	remoteCLI, err := remoteClient(r.config.Host)
	if err != nil {
		return nil, err
	}
	defer remoteCLI.Close()

	target := &Image{
		Image: r.config.Image,
		Tag:   r.config.Tag,
	}

	if r.config.Image == "" || r.config.Tag == "" {
		target = src
	}

	localImage, err := summary(src.FullImageName(), localCLI, ctx)
	if err != nil {
		return nil, err
	}
	if localImage == nil {
		return nil, errors.New("Image not found: " + src.FullImageName())
	}

	if target != src {
		u.Update("Tagging Image as: " + target.FullImageName())
		err = localCLI.ImageTag(ctx, src.FullImageName(), target.FullImageName())
		if err != nil {
			return nil, err
		}
	}

	u.Update("Pushing Image to remote Docker host")
	savedImage, err := localCLI.ImageSave(ctx, []string{target.FullImageName()})
	if err != nil {
		return nil, err
	}
	_, err = remoteCLI.ImageLoad(ctx, savedImage, false)
	if err != nil {
		return nil, err
	}
	return target, nil
}

func (r *Registry) Documentation() (*docs.Documentation, error) {
	doc, err := docs.New(docs.FromConfig(&TransporterConfig{}))
	if err != nil {
		return nil, err
	}

	doc.Description("Transfer a Docker image to a remote Docker host.")

	doc.Example(`
build {
 ...
  registry {
    use "transporter" {
      host = "ssh://user@ip:port"
      image = "my-target-image-name"
      tag   = gitrefhash()
    }
  }
}
`)

	doc.Input("registry.Image")
	doc.Output("registry.Image")

	doc.SetField(
		"host",
		"the connection url to the remote Docker host",
		docs.Summary(
			"this value must be a valid SSH connection string, e.g. ssh://user@ip:port",
		),
	)

	doc.SetField(
		"image",
		"the target image name for remote host",
		docs.Summary(
			"this value can be the fully qualified name to the image.",
		),
	)

	doc.SetField(
		"tag",
		"the tag for the target image",
		docs.Summary(
			"this is added to image to provide the full image reference",
		),
	)

	return doc, nil
}

func summary(reference string, client *client.Client, ctx context.Context) (*types.ImageSummary, error) {
	filter := filters.NewArgs(filters.Arg("reference", reference))
	ops := types.ImageListOptions{
		All:     false,
		Filters: filter,
	}
	images, ok := client.ImageList(ctx, ops)
	if ok != nil {
		return nil, ok
	}

	if len(images) > 1 {
		return nil, errors.New("Multiple images found for: " + reference)
	}
	if len(images) == 0 {
		return nil, nil
	}
	return &images[0], nil
}

func localClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv)
}

func remoteClient(host string) (*client.Client, error) {
	helper, ok := connhelper.GetConnectionHelper(host)
	if ok != nil {
		return nil, ok
	}
	httpClient := &http.Client{Transport: &http.Transport{DialContext: helper.Dialer}}

	var clientOpts []client.Opt

	clientOpts = append(clientOpts,
		client.WithHTTPClient(httpClient),
		client.WithHost(helper.Host),
		client.WithDialContext(helper.Dialer),
		client.WithAPIVersionNegotiation(),
	)
	return client.NewClientWithOpts(clientOpts...)
}
