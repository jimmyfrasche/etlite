package ast

import (
	"strings"

	"github.com/jimmyfrasche/etlite/internal/internal/digital"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/internal/escape"
	"github.com/jimmyfrasche/etlite/internal/token"
)

//Name represents an SQLite name in the form of [schema.]object.
type Name struct {
	hasSchema, hasObject bool
	schema, object       token.Value
	usc, uob             string
}

//MakeName from 1 or 3 tokens represnting a valid SQLite name.
func MakeName(tokens []token.Value) (Name, error) {
	lt := len(tokens)
	if lt != 1 && lt != 3 {
		return Name{}, errint.Newf("MakeName given %d tokens")
	}
	if k := tokens[0].Kind; k != token.Literal || k != token.String {
		kind := "name"
		if lt == 1 {
			kind = "schema"
		}
		return Name{}, errint.Newf("MakeName given invalid %s %#v", kind, tokens[0])
	}
	if lt == 1 {
		return Name{
			hasObject: true,
			object:    tokens[0],
		}, nil
	}
	if !tokens[1].Literal(".") {
		return Name{}, errint.Newf("MakeName given malformed Name: %#v", tokens)
	}
	if k := tokens[2].Kind; k != token.Literal || k != token.String {
		return Name{}, errint.Newf("MakeName on schema %s given invalid name %#v", tokens[0].Value, tokens[2])
	}
	return Name{
		hasSchema: true,
		hasObject: true,
		schema:    tokens[0],
		object:    tokens[2],
	}, nil
}

//NameFromString synthesizes a schemaless Name from name.
func NameFromString(name string) Name {
	return Name{
		hasObject: true,
		object: token.Value{
			Kind:  token.Literal,
			Value: name,
		},
	}
}

//Pos reports the original position in input.
func (n Name) Pos() token.Position {
	if n.Empty() {
		return token.Position{}
	}
	if n.hasSchema {
		return n.schema.Pos()
	}
	return n.object.Pos()
}

func (n Name) Empty() bool {
	return !n.hasObject
}

//HasSchema reports whether n was created with a schema.
func (n Name) HasSchema() bool {
	return n.hasSchema
}

//SchemaToken returns the token value of the schema identifier,
//or an empty token if none defined.
func (n Name) SchemaToken() token.Value {
	return n.schema
}

//ObjectToken returns the token value of the object identifier,
//or an empty token if none defined.
func (n Name) ObjectToken() token.Value {
	return n.object
}

//Schema returns the unescaped schema identifier of n.
func (n Name) Schema() string {
	if n.usc != "" {
		return n.usc
	}
	if !n.hasSchema {
		return ""
	}
	n.usc, _ = n.schema.Unescape()
	return n.usc
}

//Object returns the unescaped object identifier of n.
func (n Name) Object() string {
	if n.uob != "" {
		return n.uob
	}
	if !n.hasObject {
		return ""
	}
	n.uob, _ = n.object.Unescape()
	return n.uob
}

func (n Name) schemaIs(lit string) bool {
	return n.hasSchema && strings.ToLower(n.Schema()) == lit
}

//OnSys reports whether the schema identifier, if present, is for the sys database.
func (n Name) OnSys() bool {
	return n.schemaIs("sys")
}

//OnTemp reports whether the schema identifier, if present, is for the temp database.
func (n Name) OnTemp() bool {
	return n.schemaIs("temp")
}

//DigitalObject reports whether the object identifier
//consists solely of ASCII digits.
func (n Name) DigitalObject() bool {
	return n.hasObject && digital.String(n.Object())
}

//Reserved reports whether n is sys.args or sys.env.
func (n Name) Reserved() bool {
	if !n.hasObject || !n.OnSys() {
		return false
	}
	s := strings.ToLower(n.Object())
	return s == "args" || s == "env"
}

//WithoutSchema returns a new Name without a schema.
func (n Name) WithoutSchema() Name {
	return Name{
		hasObject: n.hasObject,
		object:    n.object,
		uob:       n.uob,
	}
}

//String returns the fully formatted and always escaped serialization of n.
func (n Name) String() string {
	if !n.hasObject {
		return ""
	}
	obj := escape.String(n.Object())
	if !n.hasSchema {
		return obj
	}
	return escape.String(n.Schema()) + "." + obj
}
