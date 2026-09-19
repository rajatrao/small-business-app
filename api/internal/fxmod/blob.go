package fxmod

import (
	"github.com/rajat/localdiscovery/internal/adapter/blob"
	"github.com/rajat/localdiscovery/internal/ports"
	"go.uber.org/fx"
)

func NewBlob(cfg Config) (*blob.Disk, error) {
	return blob.NewDisk(cfg.BlobDir, cfg.PublicBaseURL)
}

func asBlob(d *blob.Disk) ports.BlobStore { return d }

var BlobModule = fx.Module("blob",
	fx.Provide(NewBlob, asBlob),
)
