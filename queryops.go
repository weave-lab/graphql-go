package graphql

import (
	"errors"

	"github.com/weave-lab/graphql-go/internal/common"
	"github.com/weave-lab/graphql-go/internal/exec/selected"
	"github.com/weave-lab/graphql-go/internal/query"
)

const (
	FieldSeparator = "_"
)

// QueryOp represents a summary of query's requested operations
// this may be used for field monitoring or in resolvers to avoid
// costly operations.
type QueryOp struct {
	Name      string `json:",omitempty"`
	Type      query.OperationType
	Variables map[string]string   `json:",omitempty"`
	Fields    []QueryFieldSummary `json:",omitempty"`
}

// QueryFieldSummary represents a summary of a field requested in the query.
type QueryFieldSummary struct {
	Name      string
	Arguments map[string]string `json:",omitempty"`
}

func selectionToField(sel query.Selection) query.Field {
	if field, ok := sel.(query.Field); ok {
		return field
	} else if field, ok := sel.(*query.Field); ok {
		return *field
	}
	return query.Field{}
}

func parseQueryFields(field query.Field) []QueryFieldSummary {
	args := []common.Argument(field.Arguments)
	var fieldArgs map[string]string
	if len(args) > 0 {
		fieldArgs = make(map[string]string)
		for _, arg := range args {
			fieldArgs[arg.Name.Name] = arg.Value.String()
		}
	}

	fields := make([]QueryFieldSummary, 0)
	if len(field.Selections) > 0 {
		for _, sel := range field.Selections {
			rawField := selectionToField(sel)
			subFields := parseQueryFields(rawField)
			for _, sub := range subFields {
				newArgs := sub.Arguments
				if len(fieldArgs) > 0 {
					for k, v := range fieldArgs {
						if newArgs == nil {
							newArgs = fieldArgs
							break
						}
						//duplidate names will get overwritten
						newArgs[k] = v
					}
				}
				fields = append(fields, QueryFieldSummary{
					Name:      field.Name.Name + FieldSeparator + sub.Name,
					Arguments: newArgs,
				})
			}
		}
	} else {
		fields = append(fields, QueryFieldSummary{
			Name:      field.Name.Name,
			Arguments: fieldArgs,
		})
	}

	return fields
}

func ParseQueryOps(document interface{}) ([]QueryOp, error) {
	doc, ok := document.(*query.Document)
	if !ok {
		return nil, errors.New("invalid value passed to ParseQueryOps expected a *query.Document")
	}
	ops := []*query.Operation(doc.Operations)
	qops := make([]QueryOp, len(ops))
	for i, op := range ops {
		inputs := []*common.InputValue(op.Vars)
		var args map[string]string
		if len(inputs) > 0 {
			args = make(map[string]string)
			for _, input := range inputs {
				if input != nil && input.Default != nil {
					args[input.Name.Name] = input.Default.String()
				}
			}
		}

		fields := make([]QueryFieldSummary, 0, len(op.Selections))
		for _, sel := range op.Selections {
			rawField := selectionToField(sel)
			fields = append(fields, parseQueryFields(rawField)...)
		}

		qops[i] = QueryOp{
			Name:      op.Name.Name,
			Type:      op.Type,
			Variables: args,
			Fields:    fields,
		}
	}
	return qops, nil
}

type QueryField struct {
	Name      string
	Arguments map[string]interface{}
	Subfields []selected.Selection
}

func parseResolverFields(field *selected.SchemaField) ([]QueryField, error) {
	if field == nil {
		return nil, errors.New("field must not be nil")
	}
	out := make([]QueryField, 0, len(field.Sels))
	for _, sel := range field.Sels {
		sf, ok := sel.(*selected.SchemaField) // TODO: technically this could be one of several things. Make this more robust?
		if !ok {
			return nil, errors.New("failed to parse subselection as QueryField")
		}
		out = append(out, QueryField{
			Name:      sf.Name,
			Arguments: sf.Args,
			Subfields: sf.Sels,
		})
	}
	return out, nil
}

func ParseSubfields(field interface{}) ([]QueryField, error) {
	f, ok := field.(*selected.SchemaField)
	if !ok {
		return nil, errors.New("invalid value passed to ParseCurrentField expected a *query.Field")
	}
	return parseResolverFields(f)
}
