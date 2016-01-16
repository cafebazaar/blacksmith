package cloudconfig

import (
	"sync"
)

func init() {
	bootParamsRepo = &bootParamsDataSource{nil, &sync.Mutex{}, nil, nil}
}
