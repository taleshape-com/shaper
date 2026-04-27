// SPDX-License-Identifier: MPL-2.0

package util

import (
	"database/sql/driver"
	"os"

	"github.com/duckdb/duckdb-go/v2"
)

type GetEnvFunc struct{}

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
