# Plugin Validation

The binary must pass the CLI's `infracost plugin validate <binary>` checks:

1. Start with the shared handshake and dispense key.
2. Return valid parser identity from `GetPluginInfo`.
3. Return project type `expectedload` and default priority 0 from
   `GetParserConfig`.
4. Serve `IdentifyProjects` and `Parse` through `ParserService`.

The fixture suite covers identification, parsing across every supported
syntax, tree output, diagnostics, and ignored vendor directories. Validation
should be run against the current CLI or the `cli-5fb` review branch, both of
which implement this same contract.
