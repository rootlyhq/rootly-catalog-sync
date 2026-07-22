package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	jsonnet "github.com/google/go-jsonnet"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\(([^)]+)\)`)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	content := envVarPattern.ReplaceAllStringFunc(string(data), func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		return os.Getenv(varName)
	})

	jsonContent, err := toJSON(path, content)
	if err != nil {
		return nil, err
	}

	var versionProbe struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(jsonContent, &versionProbe); err != nil {
		return nil, fmt.Errorf("reading config version: %w", err)
	}

	switch versionProbe.Version {
	case 0, 1:
		return loadV1(jsonContent)
	case 2:
		return loadV2(jsonContent)
	default:
		return nil, fmt.Errorf("unsupported config version: %d", versionProbe.Version)
	}
}

func toJSON(path, content string) ([]byte, error) {
	ext := filepath.Ext(path)
	switch ext {
	case ".jsonnet":
		vm := jsonnet.MakeVM()
		jsonStr, err := vm.EvaluateAnonymousSnippet(path, content)
		if err != nil {
			return nil, fmt.Errorf("evaluating jsonnet: %w", err)
		}
		return []byte(jsonStr), nil
	case ".hcl":
		return convertHCLToJSON(path, content)
	default:
		var raw any
		if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
		return json.Marshal(raw)
	}
}

func loadV1(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing v1 config: %w", err)
	}
	return &cfg, nil
}

func convertHCLToJSON(filename, content string) ([]byte, error) {
	file, diags := hclsyntax.ParseConfig([]byte(content), filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing hcl: %s", diags.Error())
	}

	val, diags := file.Body.(*hclsyntax.Body).JustAttributes()
	_ = val
	_ = diags

	ctx := &hcl.EvalContext{}

	// Try v2 schema first (sync blocks), fall back to v1 (pipeline blocks).
	bodyContent, diags := file.Body.Content(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "version"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "sync"},
		},
	})
	if !diags.HasErrors() && len(bodyContent.Blocks) > 0 {
		return convertHCLV2ToJSON(bodyContent, ctx)
	}

	// Re-parse with v1 schema.
	file2, _ := hclsyntax.ParseConfig([]byte(content), filename, hcl.Pos{Line: 1, Column: 1})
	bodyContent, diags = file2.Body.Content(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "version"},
			{Name: "sync_id"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "pipeline"},
		},
	})
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing hcl schema: %s", diags.Error())
	}

	result := make(map[string]any)

	for _, attr := range bodyContent.Attributes {
		v, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return nil, fmt.Errorf("evaluating %s: %s", attr.Name, diags.Error())
		}
		result[attr.Name] = ctyToGo(v)
	}

	var pipelines []any
	for _, block := range bodyContent.Blocks {
		p, err := hclBlockToMap(block, ctx)
		if err != nil {
			return nil, err
		}
		pipelines = append(pipelines, p)
	}
	result["pipelines"] = pipelines

	return json.Marshal(result)
}

func convertHCLV2ToJSON(bodyContent *hcl.BodyContent, ctx *hcl.EvalContext) ([]byte, error) {
	result := make(map[string]any)

	for _, attr := range bodyContent.Attributes {
		v, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return nil, fmt.Errorf("evaluating %s: %s", attr.Name, diags.Error())
		}
		result[attr.Name] = ctyToGo(v)
	}

	var syncs []any
	for _, block := range bodyContent.Blocks {
		s, err := hclBlockToMap(block, ctx)
		if err != nil {
			return nil, err
		}
		syncs = append(syncs, s)
	}
	result["sync"] = syncs

	return json.Marshal(result)
}

func hclBlockToMap(block *hcl.Block, ctx *hcl.EvalContext) (map[string]any, error) {
	return hclBodyToMap(block.Body, ctx)
}

func hclBodyToMap(body hcl.Body, ctx *hcl.EvalContext) (map[string]any, error) {
	result := make(map[string]any)

	syntaxBody, ok := body.(*hclsyntax.Body)
	if !ok {
		attrs, _ := body.JustAttributes()
		for name, attr := range attrs {
			v, d := attr.Expr.Value(ctx)
			if d.HasErrors() {
				return nil, fmt.Errorf("evaluating %s: %s", name, d.Error())
			}
			result[name] = ctyToGo(v)
		}
		return result, nil
	}

	for name, attr := range syntaxBody.Attributes {
		v, d := attr.Expr.Value(ctx)
		if d.HasErrors() {
			return nil, fmt.Errorf("evaluating %s: %s", name, d.Error())
		}
		result[name] = ctyToGo(v)
	}

	for _, block := range syntaxBody.Blocks {
		child, err := hclBodyToMap(block.Body, ctx)
		if err != nil {
			return nil, err
		}

		blockType := block.Type
		if blockType == "source" || blockType == "output" {
			key := blockType + "s"
			if existing, ok := result[key]; ok {
				result[key] = append(existing.([]any), child)
			} else {
				result[key] = []any{child}
			}
		} else {
			result[blockType] = child
		}
	}

	return result, nil
}

func ctyToGo(v cty.Value) any {
	if !v.IsKnown() || v.IsNull() {
		return nil
	}
	t := v.Type()
	switch {
	case t == cty.String:
		return v.AsString()
	case t == cty.Number:
		bf := v.AsBigFloat()
		if bf.IsInt() {
			i, _ := bf.Int64()
			return i
		}
		f, _ := bf.Float64()
		return f
	case t == cty.Bool:
		return v.True()
	case t.IsListType() || t.IsTupleType() || t.IsSetType():
		var items []any
		for it := v.ElementIterator(); it.Next(); {
			_, elem := it.Element()
			items = append(items, ctyToGo(elem))
		}
		return items
	case t.IsMapType() || t.IsObjectType():
		m := make(map[string]any)
		for it := v.ElementIterator(); it.Next(); {
			k, elem := it.Element()
			m[k.AsString()] = ctyToGo(elem)
		}
		return m
	default:
		raw, err := ctyjson.Marshal(v, v.Type())
		if err != nil {
			return v.GoString()
		}
		return json.RawMessage(raw)
	}
}

const (
	KindReference = "reference"
	KindText      = "text"
	KindService   = "service"
)

var validFieldKinds = map[string]bool{
	KindText: true, "number": true, "boolean": true, KindReference: true,
	KindService: true, "group": true, "functionality": true, "environment": true,
	"incident_type": true, "cause": true, "user": true,
	"slack_channel": true, "slack_alias": true,
}

func validFieldKind(kind string) bool {
	return validFieldKinds[kind]
}

func Validate(cfg *Config) error {
	if cfg.Version != 1 && cfg.Version != 2 {
		return fmt.Errorf("unsupported config version: %d (expected 1 or 2)", cfg.Version)
	}
	if cfg.SyncID == "" {
		return fmt.Errorf("sync_id is required")
	}
	if len(cfg.Pipelines) == 0 {
		return fmt.Errorf("at least one pipeline is required")
	}
	for i, p := range cfg.Pipelines {
		if len(p.Sources) == 0 {
			return fmt.Errorf("pipeline[%d]: at least one source is required", i)
		}
		if len(p.Outputs) == 0 {
			return fmt.Errorf("pipeline[%d]: at least one output is required", i)
		}
		for j, o := range p.Outputs {
			isNative := o.Type != "" && o.Type != "catalog"
			if !isNative && o.Catalog == "" {
				return fmt.Errorf("pipeline[%d].outputs[%d]: catalog is required", i, j)
			}
			if o.ExternalID == "" {
				return fmt.Errorf("pipeline[%d].outputs[%d]: external_id is required", i, j)
			}
			if o.Name == "" {
				return fmt.Errorf("pipeline[%d].outputs[%d]: name is required", i, j)
			}
			for slug, fv := range o.Fields {
				if fv.Value == "" {
					return fmt.Errorf("pipeline[%d].outputs[%d].fields[%s]: value is required", i, j, slug)
				}
				if fv.Kind != "" && !validFieldKind(fv.Kind) {
					return fmt.Errorf("pipeline[%d].outputs[%d].fields[%s]: unsupported kind %q (valid: text, number, boolean, reference, service, group, functionality, environment, incident_type, cause, user, slack_channel, slack_alias)", i, j, slug, fv.Kind)
				}
				if fv.Kind == KindReference && fv.Catalog == "" {
					return fmt.Errorf("pipeline[%d].outputs[%d].fields[%s]: catalog is required when kind is reference", i, j, slug)
				}
			}
		}
	}
	return nil
}
