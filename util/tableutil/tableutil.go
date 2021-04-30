package tableutil

import (
	"github.com/pingcap/parser/model"
	"github.com/pingcap/tidb/meta/autoid"
)

type TempTable interface {
	GetAllocator() autoid.Allocator
	SetModified(bool)
	GetModified() bool
}

var TempTableFromMeta func (tblInfo *model.TableInfo) TempTable



