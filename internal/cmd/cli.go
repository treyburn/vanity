package cmd

import "errors"

type CLI struct{}

// TODO - make this a custom type which implements https://github.com/square/exit
var ErrValidationFailed = errors.New("validation failed")
