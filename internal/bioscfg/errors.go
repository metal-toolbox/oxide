package bioscfg

import "errors"

var (
	errInvalidConditionParams = errors.New("invalid condition parameters")
	errTaskConv               = errors.New("error in generic Task conversion")
	errUnsupportedAction      = errors.New("unsupported action")
)
