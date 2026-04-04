package cmd

import "errors"

type CLI struct{}

var ErrValidationFailed = errors.New("validation failed")
