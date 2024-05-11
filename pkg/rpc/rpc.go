package rpc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjc5/kit/pkg/fsutil"
	"github.com/tkrajina/typescriptify-golang-structs/typescriptify"
)

type RouteDef struct {
	Key    string
	Type   Type
	Input  any
	Output any
}

type Type string
type Procedure string

const (
	TypeQuery    Type = "query"
	TypeMutation Type = "mutation"
)

type Opts struct {
	OutDest   string
	RouteDefs []RouteDef
}

func GenerateTypeScript(opts Opts) error {
	err := fsutil.EnsureDir(opts.OutDest)
	if err != nil {
		return errors.New("failed to ensure out dest dir: " + err.Error())
	}
	queryTS := "\nconst queryAPIDefs = ["
	mutationTS := "\nconst mutationAPIDefs = ["

	for _, routeDef := range opts.RouteDefs {
		inputStr := ""
		outputStr := ""

		var tsToMutate = &queryTS
		if routeDef.Type == TypeMutation {
			tsToMutate = &mutationTS
		}

		if routeDef.Input != nil {
			err = makeTSStr(&inputStr, routeDef.Input)
			if err != nil {
				return errors.New("failed to convert input to ts: " + err.Error())
			}
		} else {
			inputStr = "undefined"
		}

		if routeDef.Output != nil {
			err = makeTSStr(&outputStr, routeDef.Output)
			if err != nil {
				return errors.New("failed to convert output to ts: " + err.Error())
			}
		} else {
			outputStr = "undefined"
		}

		*tsToMutate += "\n{\n" + `key: "` + routeDef.Key + `",`
		*tsToMutate += "\n" + `input: "" as unknown as ` + inputStr + ","
		*tsToMutate += "\n" + `output: "" as unknown as ` + outputStr + ","
		*tsToMutate += "\n},"
	}

	tsEnd := "\n] as const;"
	queryTS += tsEnd
	mutationTS += tsEnd
	ts := queryTS + "\n" + mutationTS + "\n"

	ts += "\n" + extraCode
	ts = "/*\n * This file is auto-generated. Do not edit.\n */\n" + ts

	err = os.WriteFile(filepath.Join(opts.OutDest, "api-types.ts"), []byte(ts), os.ModePerm)
	if err != nil {
		return errors.New("failed to write ts file: " + err.Error())
	}

	return nil
}

func makeTSStr(target *string, t any) error {
	converter := newConverter()
	converter.Add(t)

	ts, err := converter.Convert(make(map[string]string))
	if err != nil {
		return errors.New("failed to convert to ts: " + err.Error())
	}

	inputLines := strings.Split(ts, "\n")
	if len(inputLines) > 2 {
		ts = strings.Join(inputLines[2:], "\n")
		ts = "{\n" + ts
	}

	*target = ts
	return nil
}

var extraCode = `export type QueryAPIRoute = (typeof queryAPIDefs)[number];
export type MutationAPIRoute = (typeof mutationAPIDefs)[number];

export type QueryAPIKey = QueryAPIRoute["key"];
export type MutationAPIKey = MutationAPIRoute["key"];

export type QueryAPIRoutes = {
  [K in QueryAPIKey]: Extract<QueryAPIRoute, { key: K }>;
};
export type MutationAPIRoutes = {
  [K in MutationAPIKey]: Extract<MutationAPIRoute, { key: K }>;
};

export const queryAPIRoutes = Object.fromEntries(
  queryAPIDefs.map((r) => [r.key, r]),
) as QueryAPIRoutes;
export const mutationAPIRoutes = Object.fromEntries(
  mutationAPIDefs.map((r) => [r.key, r]),
) as MutationAPIRoutes;

export type QueryAPIInput<T extends QueryAPIKey> = Extract<
  QueryAPIRoute,
  { key: T }
>["input"];
export type QueryAPIOutput<T extends QueryAPIKey> = Extract<
  QueryAPIRoute,
  { key: T }
>["output"];

export type MutationAPIInput<T extends MutationAPIKey> = Extract<
  MutationAPIRoute,
  { key: T }
>["input"];
export type MutationAPIOutput<T extends MutationAPIKey> = Extract<
  MutationAPIRoute,
  { key: T }
>["output"];
`

func newConverter() *typescriptify.TypeScriptify {
	converter := typescriptify.New()
	converter.CreateInterface = true
	return converter
}
