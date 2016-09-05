//Package rawfmt defines encoding and decoding of the raw format.
//
//Raw is similar to CSV, but uses \t for the default delimiter and
//offers no means of quoting values that contain the delimiter.
//
//Unlike CSV, it defaults to not having a header.
//
//It is primarily useful as it is the default format for many command line utilities.
package rawfmt
