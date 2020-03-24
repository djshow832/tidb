// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package stmtsummary

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pingcap/tidb/config"
)

const (
	TypeEnable = iota
	TypeRefreshInterval
	TypeHistorySize
	TypeMaxStmtCount
	TypeMaxSQLLength
	TypesNum
)

type systemVars struct {
	sync.Mutex
	// This array itself won't be modified once created. Only its elements may be modified.
	variables []variable
}

type variable struct {
	sessionValue string
	globalValue  string
	finalValue   int64
}

func newSysVars() *systemVars {
	s := &systemVars{
		variables: make([]variable, TypesNum),
	}
	// Initialize these configurations by values in the config file.
	// They may be overwritten by system variables later.
	for varType := range s.variables {
		atomic.StoreInt64(&s.variables[varType].finalValue, getConfigValue(varType))
	}
	return s
}

func (s *systemVars) getVariable(varType int) int64 {
	return atomic.LoadInt64(&s.variables[varType].finalValue)
}

func (s *systemVars) setVariable(varType int, valueStr string, isSession bool) {
	s.Lock()
	defer s.Unlock()

	v := &s.variables[varType]
	if isSession {
		v.sessionValue = valueStr
	} else {
		v.globalValue = valueStr
	}
	sessionValue := v.sessionValue
	globalValue := v.globalValue

	var valueInt int64
	switch varType {
	case TypeEnable:
		valueInt = getBoolFinalVariable(varType, sessionValue, globalValue)
	case TypeHistorySize, TypeMaxSQLLength:
		valueInt = getIntFinalVariable(varType, sessionValue, globalValue, 0)
	case TypeRefreshInterval, TypeMaxStmtCount:
		valueInt = getIntFinalVariable(varType, sessionValue, globalValue, 1)
	default:
		panic(fmt.Sprintf("No such type of variable: %d", varType))
	}
	atomic.StoreInt64(&v.finalValue, valueInt)
}

func getBoolFinalVariable(varType int, sessionValue, globalValue string) int64 {
	var valueInt int64
	if len(sessionValue) > 0 {
		valueInt = normalizeEnableValue(sessionValue)
	} else if len(globalValue) > 0 {
		valueInt = normalizeEnableValue(globalValue)
	} else {
		valueInt = getConfigValue(varType)
	}
	return valueInt
}

// normalizeEnableValue converts 'ON' or '1' to 1 and 'OFF' or '0' to 0.
func normalizeEnableValue(value string) int64 {
	switch {
	case strings.EqualFold(value, "ON"):
		return 1
	case value == "1":
		return 1
	default:
		return 0
	}
}

func getIntFinalVariable(varType int, sessionValue, globalValue string, minValue int64) int64 {
	valueInt := int64(-1)
	var err error
	if len(sessionValue) > 0 {
		valueInt, err = strconv.ParseInt(sessionValue, 10, 64)
		if err != nil {
			valueInt = -1
		}
	}
	if valueInt < 0 || valueInt < minValue {
		valueInt, err = strconv.ParseInt(globalValue, 10, 64)
		if err != nil {
			valueInt = -1
		}
	}
	// If session and global variables are both '', use the value in config.
	if valueInt < 0 || valueInt < minValue {
		valueInt = getConfigValue(varType)
	}
	return valueInt
}

func getConfigValue(varType int) int64 {
	var valueInt int64
	stmtSummaryConfig := config.GetGlobalConfig().StmtSummary
	switch varType {
	case TypeEnable:
		if stmtSummaryConfig.Enable {
			valueInt = 1
		}
	case TypeRefreshInterval:
		valueInt = int64(stmtSummaryConfig.RefreshInterval)
	case TypeHistorySize:
		valueInt = int64(stmtSummaryConfig.HistorySize)
	case TypeMaxStmtCount:
		valueInt = int64(stmtSummaryConfig.MaxStmtCount)
	case TypeMaxSQLLength:
		valueInt = int64(stmtSummaryConfig.MaxSQLLength)
	default:
		panic(fmt.Sprintf("No such type of variable: %d", varType))
	}
	return valueInt
}
