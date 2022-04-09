package session_states

import (
	"time"

	"github.com/pingcap/tidb/parser/model"
	ptypes "github.com/pingcap/tidb/parser/types"
	"github.com/pingcap/tidb/types"
)

type TxnIsolationLevelOneShot struct {
	State uint   `json:"state"`
	Value string `json:"value"`
}

type SQLWarn struct {
	Level   string `json:"level"`
	Code    uint16 `json:"code"`
	Message string `json:"message"`
}

type SessionStates struct {
	LockedTables         map[int64]model.TableLockTpInfo `json:"locked-tables,omitempty"`
	UserVars             map[string]types.DatumJSON      `json:"user-vars,omitempty"`
	UserVarTypes         map[string]*ptypes.FieldType    `json:"user-var-types,omitempty"`
	SystemVars           map[string]string               `json:"system-vars,omitempty"`
	PreparedStmts        map[uint32]interface{}          `json:"prepared-stmts,omitempty"`
	PreparedStmtNameToID map[string]uint32               `json:"prepared-stmt-names,omitempty"`
	PreparedStmtID       uint32                          `json:"prepared-stmt-id,omitempty"`
	TxnIsolationLevel    *TxnIsolationLevelOneShot       `json:"txn_iso_level,omitempty"`
	Status               uint16                          `json:"status,omitempty"`
	CurrentDB            string                          `json:"current-db,omitempty"`
	LastFoundRows        uint64                          `json:"last-found-rows,omitempty"`
	LastInsertID         uint64                          `json:"last-insert-id,omitempty"`
	PrevAffectedRows     int64                           `json:"affected-rows,omitempty"`
	SequenceLatestValues map[int64]int64                 `json:"sequence-values,omitempty"`
	MPPStoreLastFailTime map[string]time.Time            `json:"store-last-fail,omitempty"`
	Warnings             []SQLWarn                       `json:"warnings,omitempty"`
}

func (ss *SessionStates) DecodeUserVars() (map[string]types.Datum, error) {
	result := make(map[string]types.Datum, len(ss.UserVars))
	for name, datumJson := range ss.UserVars {
		datum, err := types.DatumJSONToDatum(&datumJson)
		if err != nil {
			return nil, err
		}
		result[name] = *datum
	}
	return result, nil
}

func (ss *SessionStates) EncodeUserVars(userVars map[string]types.Datum) error {
	ss.UserVars = make(map[string]types.DatumJSON, len(userVars))
	for name, datum := range userVars {
		datumJson, err := types.DatumToDatumJSON(&datum)
		if err != nil {
			return err
		}
		ss.UserVars[name] = *datumJson
	}
	return nil
}

func (ss *SessionStates) DecodeUserVarFields() map[string]*ptypes.FieldType {
	result := make(map[string]*ptypes.FieldType, len(ss.UserVarTypes))
	for name, userVar := range ss.UserVarTypes {
		result[name] = userVar.Clone()
	}
	return result
}

func (ss *SessionStates) EncodeUserVarFields(userVarFields map[string]*ptypes.FieldType) {
	ss.UserVarTypes = make(map[string]*ptypes.FieldType, len(userVarFields))
	for name, userVar := range userVarFields {
		ss.UserVarTypes[name] = userVar.Clone()
	}
}
