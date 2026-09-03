# Parser Plugin Interface

The binary implements the generic `infracost.plugin` contract documented by
`parser-plugin-sdk` and used by the CLI.

- Handshake: `INFRACOST_PLUGIN=de8c7e96-497c-4168-80c4-fc875c8ce764`.
- Dispense key: `plugin`.
- Register `PluginService` and `ParserService` on the same gRPC server.
- `GetPluginInfo` reports `infracost/expectedload`, its build version, and type
  `PARSER`.
- `GetParserConfig` reports priority 45 and project type `expectedload`.
- `IdentifyProjects` claims a directory only when a supported top-level source
  file contains an expected-load marker.
- `Parse` returns diagnostics and a provider-agnostic `tree.Tree`.
- Use a tagged `github.com/infracost/proto` release. No feature-branch
  pseudo-version or `replace` directive is required.

Parser and provider plugins share this handshake and discovery service. The
type returned by `GetPluginInfo` determines which typed service the CLI uses.
