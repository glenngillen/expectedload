# Parser Plugin Interface

The binary implements the generic `infracost.plugin` contract documented by
`parser-plugin-sdk` and used by the CLI.

- Handshake: `INFRACOST_PLUGIN=de8c7e96-497c-4168-80c4-fc875c8ce764`.
- Dispense key: `plugin`.
- Register `PluginService` and `ParserService` on the same gRPC server.
- `GetPluginInfo` reports `glenngillen/expectedload`, its build version, and type
  `PARSER`.
- `GetParserConfig` reports the default priority 0 and project type
  `expectedload`.
- `IdentifyProjects` returns marker-bearing top-level source files as individual
  projects. It does not claim the whole directory, so annotations can coexist
  with directory-oriented Terraform and other parsers.
- `Parse` returns diagnostics and a provider-agnostic `tree.Tree`.
- Use a tagged `github.com/infracost/proto` release. No feature-branch
  pseudo-version or `replace` directive is required.

Parser and provider plugins share this handshake and discovery service. The
type returned by `GetPluginInfo` determines which typed service the CLI uses.
