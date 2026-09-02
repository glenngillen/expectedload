package main

import (
	"fmt"
	"path"
	"sort"
	"strconv"

	"github.com/infracost/expectedload"
	"github.com/infracost/expectedload/internal/scan"
	"github.com/infracost/proto/gen/go/infracost/parser"
	"github.com/infracost/proto/gen/go/infracost/parser/cloudformation"
	"github.com/infracost/proto/gen/go/infracost/parser/hcl"
	"github.com/infracost/proto/gen/go/infracost/parser/terraform"
	"github.com/infracost/proto/gen/go/infracost/tree"
)

// resourceType is the type identifier expected-load declarations carry in
// every output shape, so downstream consumers recognize them uniformly.
const resourceType = "expected_load"

// rebase prefixes a scan-root-relative file path so it becomes relative to
// repo_directory.
func rebase(prefix, file string) string {
	if prefix == "" {
		return file
	}
	return path.Join(prefix, file)
}

// sourceRange points at the declaration's marker line.
func sourceRange(file string, line int) *parser.SourceRange {
	return &parser.SourceRange{
		Filename:  file,
		StartLine: int64(line),
		EndLine:   int64(line),
	}
}

// declID uniquely identifies a declaration site within one parse.
func declID(file string, line int) string {
	return fmt.Sprintf("%s.%s:%d", resourceType, file, line)
}

// metaFields lists the promoted (non-numeric) model fields as key/value string
// pairs, omitting unset ones. Version is always present (it defaults to 1).
func metaFields(load *expectedload.ExpectedLoad) [][2]string {
	out := [][2]string{{"version", strconv.Itoa(load.Version)}}
	for _, kv := range [][2]string{
		{"confidence", load.Confidence},
		{"last_updated", load.LastUpdated},
		{"source", load.Source},
	} {
		if kv[1] != "" {
			out = append(out, kv)
		}
	}
	return out
}

// sortedFields returns the numeric load fields in deterministic order.
func sortedFields(load *expectedload.ExpectedLoad) []string {
	keys := make([]string, 0, len(load.Fields))
	for k := range load.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// toProtoDiagnostics maps the scan's library diagnostics onto the SDK shape:
// library Warning → warning severity, library Error → error severity; message,
// offending field, and file/line survive the translation.
func toProtoDiagnostics(diags []scan.Diagnostic, prefix string) []*parser.Diagnostic {
	out := make([]*parser.Diagnostic, 0, len(diags))
	for _, d := range diags {
		pd := &parser.Diagnostic{
			Error:   d.Message,
			Warning: d.Severity == expectedload.Warning,
		}
		if d.ReadFailure {
			pd.Type = parser.DiagnosticType_DIAGNOSTIC_TYPE_FILESYSTEM_ERROR
		}
		if d.Field != "" {
			pd.Labels = map[string]string{"field": d.Field}
		}
		if d.File != "" {
			pd.SourceRange = sourceRange(rebase(prefix, d.File), d.Line)
		}
		out = append(out, pd)
	}
	return out
}

// toTree converts scanned declarations into the provider-agnostic tree — the
// primary output path. One resource per declaration site; load fields and meta
// fields ride in the resource attributes, the site in Definition.Source.
func toTree(res scan.Result, prefix string) *tree.Tree {
	svc := &tree.Service{}
	for _, d := range res.Declarations {
		file := rebase(prefix, d.File)
		entries := map[string]*tree.Value{}
		for _, k := range sortedFields(d.Load) {
			name := k
			entries[k] = &tree.Value{
				SourceFieldName: &name,
				Source:          sourceRange(file, d.Line),
				Value:           &tree.Value_IntValue{IntValue: d.Load.Fields[k]},
			}
		}
		for _, kv := range metaFields(d.Load) {
			name := kv[0]
			entries[kv[0]] = &tree.Value{
				SourceFieldName: &name,
				Source:          sourceRange(file, d.Line),
				Value:           &tree.Value_StringValue{StringValue: kv[1]},
			}
		}

		svc.Resources = append(svc.Resources, &tree.Resource{
			Id:   declID(file, d.Line),
			Type: resourceType,
			Definition: &tree.Definition{
				Source:       sourceRange(file, d.Line),
				ResourceType: resourceType,
			},
			// Declarations are usage annotations, not resources the provider
			// plugins can cost on their own.
			IsSupported: false,
			IsFree:      true,
			Attributes:  &tree.ValueObject{Entries: entries},
		})
	}

	return &tree.Tree{
		Providers: map[string]*tree.Provider{
			"expectedload": {Services: map[string]*tree.Service{"expected-load": svc}},
		},
	}
}

// toTerraformResult renders declarations into the terraform result variant:
// one resource per site, load fields as an HCL object value.
func toTerraformResult(res scan.Result, prefix string) *terraform.ModuleResult {
	out := &terraform.ModuleResult{}
	for _, d := range res.Declarations {
		file := rebase(prefix, d.File)
		mv := &hcl.MapValue{IsObject: true}
		for _, k := range sortedFields(d.Load) {
			mv.Entries = append(mv.Entries, &hcl.MapEntry{Key: k, Value: &hcl.Value{
				Value: &hcl.Value_Primitive{Primitive: &hcl.PrimitiveValue{
					Value: &hcl.PrimitiveValue_NumberValue{NumberValue: strconv.FormatInt(d.Load.Fields[k], 10)},
				}},
			}})
		}
		for _, kv := range metaFields(d.Load) {
			mv.Entries = append(mv.Entries, &hcl.MapEntry{Key: kv[0], Value: &hcl.Value{
				Value: &hcl.Value_Primitive{Primitive: &hcl.PrimitiveValue{
					Value: &hcl.PrimitiveValue_StringValue{StringValue: kv[1]},
				}},
			}})
		}

		out.Resources = append(out.Resources, &terraform.Resource{
			Id:          declID(file, d.Line),
			Name:        declID(file, d.Line),
			Type:        resourceType,
			SourceRange: sourceRange(file, d.Line),
			Supported:   false,
			Free:        true,
			Data:        &hcl.Value{Value: &hcl.Value_Map{Map: mv}},
		})
	}
	return out
}

// toCloudformationResult renders declarations into the cloudformation result
// variant: one resource per site, load fields in the metadata map.
func toCloudformationResult(res scan.Result, prefix string) *cloudformation.Result {
	out := &cloudformation.Result{Resources: map[string]*cloudformation.Resource{}}
	for _, d := range res.Declarations {
		file := rebase(prefix, d.File)
		meta := map[string]*cloudformation.Value{}
		for _, k := range sortedFields(d.Load) {
			meta[k] = &cloudformation.Value{Value: &cloudformation.Value_Scalar{Scalar: &cloudformation.ScalarValue{
				Type:  cloudformation.ScalarType_SCALAR_TYPE_INT,
				Value: &cloudformation.ScalarValue_IntValue{IntValue: d.Load.Fields[k]},
			}}}
		}
		for _, kv := range metaFields(d.Load) {
			meta[kv[0]] = &cloudformation.Value{Value: &cloudformation.Value_Scalar{Scalar: &cloudformation.ScalarValue{
				Type:  cloudformation.ScalarType_SCALAR_TYPE_STRING,
				Value: &cloudformation.ScalarValue_StringValue{StringValue: kv[1]},
			}}}
		}

		id := declID(file, d.Line)
		out.Resources[id] = &cloudformation.Resource{
			Id:          id,
			Type:        resourceType,
			Metadata:    meta,
			SourceRange: sourceRange(file, d.Line),
			Supported:   false,
			Free:        true,
		}
	}
	return out
}
