package registry

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/hashicorp/waypoint-plugin-sdk/component"
	"github.com/hashicorp/waypoint-plugin-sdk/terminal"
	"net/http"
)

type TransporterConfig struct {
	Host     string `hcl:"host"`
	Image    string `hcl:"image"`
	Tag      string `hcl:"tag"`
	Override bool   `hcl:"override_existing,optional"`
	CheckID  bool   `hcl:"compare_checksum,optional"`
}

type Registry struct {
	config TransporterConfig
}

// Implement Configurable
func (r *Registry) Config() (interface{}, error) {
	return &r.config, nil
}

// Implement ConfigurableNotify
func (r *Registry) ConfigSet(config interface{}) error {
	c, ok := config.(*TransporterConfig)
	if !ok {
		// The Waypoint SDK should ensure this never gets hit
		return fmt.Errorf("expected *transporterconfig as parameter")
	}

	// validate the config
	if c.Host == "" {
		return fmt.Errorf("host must be set to a valid ssh url")
	}

	if c.Image == "" {
		return fmt.Errorf("image name must be set")
	}

	if c.Tag == "" {
		return fmt.Errorf("image tag must be set")
	}

	return nil
}

// Implement Registry
func (r *Registry) PushFunc() interface{} {
	// return a function which will be called by Waypoint
	return r.push
}

// A PushFunc does not have a strict signature, you can define the parameters
// you need based on the Available parameters that the Waypoint SDK provides.
// Waypoint will automatically inject parameters as specified
// in the signature at run time.
//
// Available input parameters:
// - context.Context
// - *component.Source
// - *component.JobInfo
// - *component.DeploymentConfig
// - *datadir.Project
// - *datadir.App
// - *datadir.Component
// - hclog.Logger
// - terminal.UI
// - *component.LabelSet
//
// In addition to default input parameters the builder.Binary from the Build step
// can also be injected.
//
// The output parameters for PushFunc must be a Struct which can
// be serialzied to Protocol Buffers binary format and an error.
// This Output Value will be made available for other functions
// as an input parameter.
// If an error is returned, Waypoint stops the execution flow and
// returns an error to the user.
func (r *Registry) push(src *component.Source, ctx context.Context, ui terminal.UI) (*Image, error) {
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

	localImgRef := src.App + ":latest"
	localImage, err := summary(localImgRef, localCLI, ctx)
	if err != nil {
		return nil, err
	}
	if localImage == nil {
		return nil, errors.New("Image not found: " + localImgRef)
	}

	remoteImgRef := r.config.Image + ":" + r.config.Tag
	remoteImage, err := summary(remoteImgRef, remoteCLI, ctx)
	if err != nil {
		return nil, err
	}
	if remoteImage != nil && r.config.CheckID && localImage.ID == remoteImage.ID {
		u.Step("Nothing to do.", "Image not changed, skipping push")
		image := &Image{
			Image: r.config.Image,
			Tag:   r.config.Tag,
		}
		return image, nil
	}
	u.Update("Tagging Image: " + localImgRef + " as: " + remoteImgRef)
	err = localCLI.ImageTag(ctx, localImgRef, remoteImgRef)
	if err != nil {
		return nil, err
	}
	u.Update("Pushing Image to remote Host")
	savedImage, err := localCLI.ImageSave(ctx, []string{remoteImgRef})
	if err != nil {
		return nil, err
	}
	_, err = remoteCLI.ImageLoad(ctx, savedImage, false)
	if err != nil {
		return nil, err
	}
	return &Image{
		Image: r.config.Image,
		Tag:   r.config.Tag,
	}, nil
}

func (a *Image) Labels() map[string]string {
	return make(map[string]string)
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
