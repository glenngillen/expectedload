# Expected-Load Detection

`IdentifyProjects` receives a directory from the CLI. The plugin checks only
that directory's top-level files and claims it when a supported file contains
an expected-load marker.

Reads are bounded to 256 KB per file. Unknown, missing, and unreadable paths
return an empty response rather than an error. Recursive parsing happens later
in `Parse`; identification stays intentionally cheap.

Supported extensions are `.tf`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.go`,
`.java`, `.kt`, and `.rs`.
