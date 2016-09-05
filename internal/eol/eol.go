//Package eol defines the platform specific default end of line encoding.
package eol

//Default is the platform specific end of line encoding.
//
//On Windows, it's true, meaning "\r\n".
//
//Elsewhere, it's false, meaning "\n".
const Default bool = defaultUseCRLF
