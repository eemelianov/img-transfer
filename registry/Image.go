package registry

func (i *Image) FullImageName() string {
	return i.Image + ":" + i.Tag
}
