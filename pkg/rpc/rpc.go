package rpc

import (
	"strings"
	"text/template"

	"github.com/sjc5/kit/pkg/tsgen"
)

const ItemsArrayVarName = "routes"

type RouteDef struct {
	Path       string
	ActionType ActionType
	Input      any
	Output     any
}

type ActionType = string

const (
	ActionTypeQuery    ActionType = "query"
	ActionTypeMutation ActionType = "mutation"
)

type AdHocType = tsgen.AdHocType

type Opts struct {
	// Path, including filename, where the resulting TypeScript file will be written
	OutPath          string
	RouteDefs        []RouteDef
	AdHocTypes       []AdHocType
	ExportItemsArray bool
}

func GenerateTypeScript(opts Opts) error {
	var items []tsgen.Item

	for _, r := range opts.RouteDefs {
		items = append(items, tsgen.Item{
			ArbitraryProperties: []tsgen.ArbitraryProperty{
				{Name: "path", Value: r.Path},
				{Name: "actionType", Value: r.ActionType},
			},
			PhantomTypes: []tsgen.PhantomType{
				{PropertyName: "phantomInputType", TypeInstance: r.Input, TSTypeName: r.Path + "Input"},
				{PropertyName: "phantomOutputType", TypeInstance: r.Output, TSTypeName: r.Path + "Output"},
			},
		})
	}

	return tsgen.GenerateTSToFile(tsgen.Opts{
		OutPath:           opts.OutPath,
		AdHocTypes:        opts.AdHocTypes,
		Items:             items,
		ExtraTSCode:       extraTSCode,
		ItemsArrayVarName: ItemsArrayVarName,
		ExportItemsArray:  opts.ExportItemsArray,
	})
}

var extraTSCode = getExtraTSCode()

func getExtraTSCode() string {
	var extraTSBuilder strings.Builder

	categories := []struct {
		Prefix     string
		ActionType ActionType
	}{
		{Prefix: "Query", ActionType: ActionTypeQuery},
		{Prefix: "Mutation", ActionType: ActionTypeMutation},
	}

	for i, c := range categories {
		err := extraTSTmpl.Execute(&extraTSBuilder, map[string]string{
			"Prefix":            c.Prefix,
			"ActionType":        c.ActionType,
			"ItemsArrayVarName": ItemsArrayVarName,
		})
		if err != nil {
			panic(err)
		}

		if i == 0 {
			extraTSBuilder.WriteString("\n")
		}
	}

	return extraTSBuilder.String()
}

var extraTSTmpl = template.Must(template.New("extraTS").Parse(extraTSTmplStr))

const extraTSTmplStr = `export type {{ .Prefix }}APIRoute = Extract<(typeof {{ .ItemsArrayVarName }})[number], { actionType: "{{ .ActionType }}" }>;
export type {{ .Prefix }}APIPath = {{ .Prefix }}APIRoute["path"];
export type {{ .Prefix }}APIInput<T extends {{ .Prefix }}APIPath> = Extract<
	{{ .Prefix }}APIRoute,
	{ path: T }
>["phantomInputType"];
export type {{ .Prefix }}APIOutput<T extends {{ .Prefix }}APIPath> = Extract<
	{{ .Prefix }}APIRoute,
	{ path: T }
>["phantomOutputType"];
export type {{ .Prefix }}APIRoutes = {
	[K in {{ .Prefix }}APIPath]: Extract<{{ .Prefix }}APIRoute, { path: K }>;
};
`
