// SPDX-License-Identifier: MPL-2.0

package util

import (
	"database/sql/driver"
	"fmt"
	"os"
	"sync"

	"github.com/duckdb/duckdb-go/v2"
)

type GetEnvFunc struct {
	mu      sync.RWMutex
	enabled bool
}

func (f *GetEnvFunc) Config() duckdb.ScalarFuncConfig {
	varcharInfo, _ := duckdb.NewTypeInfo(duckdb.TYPE_VARCHAR)
	return duckdb.ScalarFuncConfig{
		InputTypeInfos: []duckdb.TypeInfo{varcharInfo},
		ResultTypeInfo: varcharInfo,
	}
}

func (f *GetEnvFunc) Executor() duckdb.ScalarFuncExecutor {
	return duckdb.ScalarFuncExecutor{
		RowExecutor: func(values []driver.Value) (any, error) {
			f.mu.RLock()
			defer f.mu.RUnlock()
			if !f.enabled {
				return nil, fmt.Errorf("getenv() is only available during initialization")
			}
			if len(values) == 0 || values[0] == nil {
				return nil, nil
			}
			key, ok := values[0].(string)
			if !ok {
				return nil, nil
			}
			return os.Getenv(key), nil
		},
	}
}

func (f *GetEnvFunc) Enable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = true
}

func (f *GetEnvFunc) Disable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = false
}
