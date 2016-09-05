//Package lex is a mostly compliant SQLite lexer.
//
//The lexer is in some ways stricter, ? and $ bindings are disallowed.
//
//This lexer also does not include unary prefixes for numbers and relies
//on the output to not include spaces between operators and numbers.
package lex
