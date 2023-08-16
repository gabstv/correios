package correios

import (
	"net/url"
	"time"
)

var (
	FallbackFunc      func(v url.Values) (*FreteResponse, error)
	GlobalTimeout     time.Duration
	AlwaysUseFallback bool
)
