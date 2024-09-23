package kind

import (
	"errors"
	"strings"
)

type Controller uint8

const (
	BiosCfg Controller = iota
)

const (
	BiosCfgStr = "bioscfg"
)

var (
	ErrUnknownControllerKind = errors.New("unknown controller kind")
)

func (c Controller) String() string {
	switch c {
	case BiosCfg:
		return BiosCfgStr
	default:
		return "unknown"
	}
}

func FromString(str string) (Controller, error) {
	switch strings.ToLower(str) {
	case BiosCfgStr:
		return BiosCfg, nil
	default:
		return 0, ErrUnknownControllerKind
	}
}
