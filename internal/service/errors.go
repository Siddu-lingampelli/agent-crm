package service

import "errors"

var ErrNotConfigured = errors.New("this operation has no configured implementation")
