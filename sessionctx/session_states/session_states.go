package session_states

import (
	"time"

	"github.com/pingcap/tidb/parser/model"
	ptypes "github.com/pingcap/tidb/parser/types"
	"github.com/pingcap/tidb/sessionctx/stmtctx"
	"github.com/pingcap/tidb/types"
)

const (
	StatePrepareStmt int = iota
	StateBinding
)

type PreparedStmtInfo struct {
	Name       string `json:"name,omitempty"`
	StmtText   string `json:"text"`
	StmtDB     string `json:"db,omitempty"`
	ParamTypes []byte `json:"types,omitempty"`
}

// QueryInfo represents the information of last executed query. It's used to expose information for test purpose.
type QueryInfo struct {
	TxnScope    string `json:"txn_scope"`
	StartTS     uint64 `json:"start_ts"`
	ForUpdateTS uint64 `json:"for_update_ts"`
	ErrMsg      string `json:"error,omitempty"`
}

// LastDDLInfo represents the information of last DDL. It's used to expose information for test purpose.
type LastDDLInfo struct {
	Query  string `json:"query"`
	SeqNum uint64 `json:"seq_num"`
}

type BindRecord struct {
	OriginalSQL string    `json:"original_sql"`
	Db          string    `json:"db,omitempty"`
	Bindings    []Binding `json:"bindings"`
}

type Binding struct {
	BindSQL    string     `json:"bind_sql"`
	Status     string     `json:"status"`
	CreateTime types.Time `json:"create_time"`
	UpdateTime types.Time `json:"update_time"`
	Source     string     `json:"source"`
	Charset    string     `json:"charset"`
	Collation  string     `json:"collation"`
}

type SessionStates struct {
	LockedTables map[int64]model.TableLockTpInfo `json:"locked-tables,omitempty"`
	// TODO: advisoryLocks
	UserVars             map[string]*types.Datum      `json:"user-var-values,omitempty"`
	UserVarTypes         map[string]*ptypes.FieldType `json:"user-var-types,omitempty"`
	SystemVars           map[string]string            `json:"sys-vars,omitempty"`
	PreparedStmts        map[uint32]*PreparedStmtInfo `json:"prepared-stmts,omitempty"`
	PreparedStmtID       uint32                       `json:"prepared-stmt-id,omitempty"`
	Bindings             []*BindRecord                `json:"bindings,omitempty"`
	Status               uint16                       `json:"status,omitempty"`
	CurrentDB            string                       `json:"current-db,omitempty"`
	LastTxnInfo          string                       `json:"txn-info,omitempty"`
	LastQueryInfo        *QueryInfo                   `json:"query-info,omitempty"`
	LastDDLInfo          *LastDDLInfo                 `json:"ddl-info,omitempty"`
	LastFoundRows        uint64                       `json:"found-rows,omitempty"`
	LastInsertID         uint64                       `json:"last-insert-id,omitempty"`
	LastAffectedRows     int64                        `json:"affected-rows,omitempty"`
	FoundInPlanCache     bool                         `json:"in-plan-cache,omitempty"`
	FoundInBinding       bool                         `json:"in-binding,omitempty"`
	SequenceLatestValues map[int64]int64              `json:"seq-values,omitempty"`
	MPPStoreLastFailTime map[string]time.Time         `json:"store-fail-time,omitempty"`
	Warnings             []stmtctx.SQLWarn            `json:"warnings,omitempty"`
}
