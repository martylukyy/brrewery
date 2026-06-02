package system

import "errors"

var ErrUnsupported = errors.New("system metrics are only available on linux")
