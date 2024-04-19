package model

import (
	"github.com/pkg/errors"
)

var (
	ErrConfig        = errors.New("configuration error")
	ErrInvalidAction = errors.New("invalid action")
)
