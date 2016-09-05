//Package conflict provides a type for the SQLite conflict resolution methods.
package conflict

import "errors"

//Method is an enumeration of the SQLite conflict resolution methods.
type Method int

const (
	//Rollback is ROLLBACK
	Rollback Method = iota
	//Abort is ABORT
	Abort
	//Fail is FAIL
	Fail
	//Ignore is IGNORE
	Ignore
	//Replace is REPLACE
	Replace
)

//Valid returns an error if m is invalid.
func (m Method) Valid() error {
	if m < Rollback || m > Replace {
		return errors.New("unknown conflict resolution method")
	}
	return nil
}

//String returns a valid SQL identifier for valid Methods.
func (m Method) String() string {
	switch m {
	case Rollback:
		return "ROLLBACK"
	case Abort:
		return "ABORT"
	case Fail:
		return "FAIL"
	case Ignore:
		return "IGNORE"
	case Replace:
		return "REPLACE"
	}
	return "<UNKNOWN CONFLICT METHOD>"
}

//On returns "ON CONFLICT " + m.String()
func (m Method) On() string {
	if m.Valid() != nil {
		return m.String()
	}
	return "ON CONFLICT " + m.String()
}
