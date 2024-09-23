package fleetdb

import "github.com/pkg/errors"

var (
	ErrSlugs                        = errors.New("slugs error")
	ErrServerServiceRegisterChanges = errors.New("error in server service API register changes")
	ErrAssetObject                  = errors.New("asset object error")
	ErrAssetObjectConversion        = errors.New("error converting asset object")
	ErrFleetDBObject                = errors.New("serverService object error")
	ErrChangeList                   = errors.New("error building change list")
	ErrServerServiceAttrObject      = errors.New("error in server service attribute object")
	ErrFleetDBConfig                = errors.New("fleetdb configuration error")
	ErrInventoryQuery               = errors.New("fleetdb query returned error")
)
