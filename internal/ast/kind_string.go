// Code generated by "stringer -type=Kind"; DO NOT EDIT

package ast

import "fmt"

const _Kind_name = "InvalidQueryExecCreateTableFromInsertUsingSavepointReleaseBeginTransactionCommit"

var _Kind_index = [...]uint8{0, 7, 12, 16, 31, 42, 51, 58, 74, 80}

func (i Kind) String() string {
	if i < 0 || i >= Kind(len(_Kind_index)-1) {
		return fmt.Sprintf("Kind(%d)", i)
	}
	return _Kind_name[_Kind_index[i]:_Kind_index[i+1]]
}
