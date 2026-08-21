/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package database

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"

	"bkauth/pkg/logging"
	"bkauth/pkg/util"
)

const (
	ArgsTruncateLength = 4096

	// mysqlErrDupEntry is ER_DUP_ENTRY, raised when a write violates a UNIQUE key.
	mysqlErrDupEntry = 1062
)

// ============== error classification ==============

// IsDuplicateEntryError reports whether err is MySQL's unique-key violation.
//
// A unique index is the only guard that holds under concurrency: an
// application-side existence check and the write that follows it are two
// statements with a gap in between, so two requests can both pass the check.
// Callers translate the violation into their own domain error rather than
// letting a lost race surface as a 500.
//
// It deliberately does not say which key was violated: the index name is only
// available by parsing the driver's message, whose format differs across MySQL
// versions (8.0 qualifies the name with the table, 5.7 does not). A caller whose
// table carries several unique keys has to establish from the statement itself
// that the outcome is unambiguous.
func IsDuplicateEntryError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDupEntry
}

// ============== tx Rollback Log ==============

// RollBackWithLog will rollback and log if error
func RollBackWithLog(tx *sqlx.Tx) {
	if tx == nil {
		return
	}
	err := tx.Rollback()
	if err != sql.ErrTxDone && err != nil {
		logging.GetSQLLogger().Sugar().Error(err)
	}
}

// ============== slow sql logger ==============
func logSlowSQL(start time.Time, query string, args interface{}) {
	elapsed := time.Since(start)
	// to ms
	latency := float64(elapsed / time.Millisecond)

	logger := logging.GetSQLLogger()

	// current, set 20ms
	if latency > 20 {
		logger.Error(
			"-",
			zap.String("sql", strings.ReplaceAll(query, "\n\t\t", " ")),
			zap.String("sql", truncateArgs(args, ArgsTruncateLength)),
			zap.Float64("latency", latency),
		)
	}
}

func truncateArgs(args interface{}, length int) string {
	s, err := jsoniter.MarshalToString(args)
	if err != nil {
		s = fmt.Sprintf("%v", args)
	}
	return util.TruncateString(s, length)
}

func isBlank(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	}

	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

// AllowBlankFields store the fields of the struct which allow blank
// NOTE: the key is the field name in the struct, not the db tag!
type AllowBlankFields struct {
	keys map[string]struct{}
}

// NewAllowBlankFields create a allow fields
func NewAllowBlankFields() AllowBlankFields {
	return AllowBlankFields{keys: map[string]struct{}{}}
}

// HasKey check if key exist in allowed fields
func (a *AllowBlankFields) HasKey(key string) bool {
	_, ok := a.keys[key]
	return ok
}

// AddKey add a key into allowed fields
func (a *AllowBlankFields) AddKey(key string) {
	a.keys[key] = struct{}{}
}

// ParseUpdateStruct parse a struct into updated fields
func ParseUpdateStruct(values interface{}, allowBlankFields AllowBlankFields) (string, map[string]interface{}, error) {
	var setFields []string
	updateData := map[string]interface{}{}

	// TODO: allowBlankFields maybe nil?

	v := reflect.ValueOf(values)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			dbField := v.Type().Field(i).Tag.Get("db")
			if dbField == "" {
				continue
			}

			name := v.Type().Field(i).Name

			value := v.FieldByName(name)
			// TODO: should not be the id? or some other field?
			if !isBlank(value) || allowBlankFields.HasKey(name) {
				setFields = append(setFields, fmt.Sprintf("%s=:%s", dbField, dbField))
				updateData[dbField] = v.FieldByName(name).Interface()
			}
		}
	}

	setExpr := strings.Join(setFields, ", ")

	return setExpr, updateData, nil
}
