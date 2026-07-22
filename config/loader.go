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

	var cfg Config
	ext := filepath.Ext(path)
	switch ext {
	case ".jsonnet":
		vm := jsonnet.MakeVM()
		jsonStr, err := vm.EvaluateAnonymousSnippet(path, content)
		if err != nil {
			return nil, fmt.Errorf("evaluating jsonnet: %w", err)
		}
		if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
			return nil, fmt.Errorf("parsing jsonnet output: %w", err)
		}
	case ".hcl":
		jsonBytes, err := convertHCLToJSON(path, content)
		if err != nil {
			return nil, fmt.Errorf("converting hcl to json: %w", err)
		}
		if err := json.Unmarshal(jsonBytes, &cfg); err != nil {
			return nil, fmt.Errorf("parsing hcl config: %w", err)
		}
	default:
		if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
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
	bodyContent, diags := file.Body.Content(&hcl.BodySchema{
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

var validFieldKinds = map[string]bool{
	"text": true, "number": true, "boolean": true, "reference": true,
	"service": true, "group": true, "functionality": true, "environment": true,
	"incident_type": true, "cause": true, "user": true,
	"slack_channel": true, "slack_alias": true,
}

func validFieldKind(kind string) bool {
	return validFieldKinds[kind]
}

func Validate(cfg *Config) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (expected 1)", cfg.Version)
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
				if fv.Kind == "reference" && fv.Catalog == "" {
					return fmt.Errorf("pipeline[%d].outputs[%d].fields[%s]: catalog is required when kind is reference", i, j, slug)
				}
			}
		}
	}
	return nil
}
