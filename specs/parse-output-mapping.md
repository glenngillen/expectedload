# Parse Output Mapping

`Parse` scans `ParseRequest.path`, which may be a directory or one file. It
selects the parser by extension, extracts expected-load blocks, and returns a
generic `tree.Tree`.

Each declaration becomes an `expected_load` resource. Numeric fields and the
`version`, `confidence`, `last_updated`, and `source` metadata are tree
attributes. The definition contains the declaration's scan-root-relative file
and marker line.

Recoverable file and declaration failures become parser diagnostics while
valid declarations remain in the response. A missing request path is an RPC
error. Vendor directories such as `node_modules`, `vendor`, `.git`, and
`.terraform` are skipped.
