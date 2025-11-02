package repositories

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var chromdpWorker = cache.New(time.Hour, time.Hour)
