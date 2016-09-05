package token

//Kind of a token.
type Kind int

const (
	//Illegal token.
	Illegal Kind = iota
	//Literal token.
	Literal
	//String (single, double, bracket, back tick, or blob) token.
	String
	//Argument is @NNN or @ENV_VAR token.
	Argument
	//Placeholder is a pseudo-token for lifting non-standard subqueries
	//out of regular sql.
	Placeholder
	//LParen is a ( token.
	LParen
	//RParen is a ) token.
	RParen
	//Semicolon is a ; token.
	Semicolon
)

//Empty is true if the kind of token has no value.
func (k Kind) Empty() bool {
	switch k {
	case LParen,
		RParen,
		Semicolon,
		Placeholder:
		return true
	}
	return false
}

func (k Kind) String() string {
	switch k {
	case Illegal:
		return "illegal"
	case Literal:
		return "literal"
	case String:
		return "string"
	case Argument:
		return "argument"
	case Placeholder:
		return "[placeholder]"
	case LParen:
		return "("
	case RParen:
		return ")"
	case Semicolon:
		return ";"
	}
	return "<UNKNOWN KIND OF TOKEN>"
}
