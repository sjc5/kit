package tsgen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjc5/kit/pkg/chirpc"
	"github.com/sjc5/kit/pkg/fsutil"
	"github.com/tkrajina/typescriptify-golang-structs/typescriptify"
)

type Opts struct {
	OutDest    string
	BackupDest string
	Routes     []chirpc.Def
}

func Get(opts Opts) error {
	err := fsutil.EnsureDir(opts.OutDest)
	if err != nil {
		return errors.New("failed to ensure out dest dir: " + err.Error())
	}
	if opts.BackupDest != "" {
		err = fsutil.EnsureDir(opts.BackupDest)
		if err != nil {
			return errors.New("failed to ensure backup dest dir: " + err.Error())
		}
	}
	ts := ""
	routeNames := make([]string, 0, len(opts.Routes))

	for _, routeDef := range opts.Routes {
		routeNames = append(routeNames, routeDef.Name)

		if routeDef.Input != nil {
			converter := newConverter()
			converter.Add(routeDef.Input)

			inputStr, err := converter.Convert(make(map[string]string))
			if err != nil {
				return errors.New("failed to convert input to ts: " + err.Error())
			}

			inputLines := strings.Split(inputStr, "\n")
			if len(inputLines) > 2 {
				inputStr = strings.Join(inputLines[2:], "\n")
				inputStr = "export type " + routeDef.Name + "Input = {\n" + inputStr
				ts += inputStr + ";\n"
			}
		} else {
			ts += "export type " + routeDef.Name + "Input = undefined;\n"
		}

		if routeDef.Output != nil {
			converter := newConverter()
			converter.Add(routeDef.Output)

			outputStr, err := converter.Convert(make(map[string]string))
			if err != nil {
				return errors.New("failed to convert output to ts: " + err.Error())
			}

			ouputLines := strings.Split(outputStr, "\n")
			if len(ouputLines) > 2 {
				outputStr = strings.Join(ouputLines[2:], "\n")
				outputStr = "export type " + routeDef.Name + "Output = {\n" + outputStr
				ts += outputStr + ";\n"
			}
		} else {
			ts += "export type " + routeDef.Name + "Output = undefined;\n"
		}

		ts += "const " + routeDef.Name + " = " + toTsRouteDef(routeDef) + "\n"
	}

	ts += "\nexport const API_ROUTE_DEFS = [" + strings.Join(routeNames, ",") + "] as const;"
	ts += "\n" + extraCode
	ts = "/*\n * This file is auto-generated. Do not edit.\n */\n" + ts

	err = os.WriteFile(filepath.Join(opts.OutDest, "api-types.ts"), []byte(ts), os.ModePerm)
	if err != nil {
		return errors.New("failed to write ts file: " + err.Error())
	}

	return nil
}

var extraCode = `
export type ApiRoute = (typeof API_ROUTE_DEFS)[number];
export type QueryApiRoute = Extract<ApiRoute, { type: "query" }>;
export type MutationApiRoute = Extract<ApiRoute, { type: "mutation" }>;

export type ApiName = ApiRoute["name"];
export type QueryApiName = QueryApiRoute["name"];
export type MutationApiName = MutationApiRoute["name"];

export type ApiRoutes = {
	[K in ApiName]: Extract<ApiRoute, { name: K }>;
};

export const API_ROUTES = Object.fromEntries(
	API_ROUTE_DEFS.map((r) => [r.name, r]),
) as ApiRoutes;

export type ApiInput<T extends ApiName> = Extract<
	ApiRoute,
	{ name: T }
>["input"];

export type ApiOutput<T extends ApiName> = Extract<
	ApiRoute,
	{ name: T }
>["output"];
`

func toTsRouteDef(routeDef chirpc.Def) string {
	return fmt.Sprintf(
		`{
  name: "%s",
  endpoint: "%s",
  type: "%s",
  input: "" as unknown as %s,
  output: "" as unknown as %s,
} as const;`,
		routeDef.Name,
		routeDef.Endpoint,
		routeDef.Type,
		routeDef.Name+"Input",
		routeDef.Name+"Output",
	)
}

func newConverter() *typescriptify.TypeScriptify {
	converter := typescriptify.New()
	converter.CreateInterface = true
	return converter
}
