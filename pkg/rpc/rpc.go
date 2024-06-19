package rpc

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

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
	prereqsMap := make(map[string]int)
	prereqs := ""
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
			namePrefix := routeDef.Key
			if namePrefix == "" {
				namePrefix = "AnonType"
			}
			name := convertToTSVariableName(namePrefix + "_input")
			locPrereqs, err := makeTSStr(&inputStr, routeDef.Input, &prereqsMap, name)
			if err != nil {
				return errors.New("failed to convert input to ts: " + err.Error())
			}
			if locPrereqs != "" {
				prereqs += locPrereqs
			}
		} else {
			inputStr = "undefined"
		}

		if routeDef.Output != nil {
			namePrefix := routeDef.Key
			if namePrefix == "" {
				namePrefix = "AnonType"
			}
			name := convertToTSVariableName(namePrefix + "_output")
			locPrereqs, err := makeTSStr(&outputStr, routeDef.Output, &prereqsMap, name)
			if err != nil {
				return errors.New("failed to convert output to ts: " + err.Error())
			}
			if locPrereqs != "" {
				prereqs += locPrereqs
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
	ts = "/*\n * This file is auto-generated. Do not edit.\n */\n" + prereqs + ts

	err = os.WriteFile(filepath.Join(opts.OutDest, "api-types.ts"), []byte(ts), os.ModePerm)
	if err != nil {
		return errors.New("failed to write ts file: " + err.Error())
	}

	return nil
}

func makeTSStr(target *string, t any, prereqsMap *map[string]int, name string) (string, error) {
	converter := newConverter()
	converter.Add(t)

	// quiet typescriptify logs
	oldStdout := os.Stdout
	null, _ := os.Open(os.DevNull)
	os.Stdout = null
	ts, err := converter.Convert(make(map[string]string))
	null.Close()
	os.Stdout = oldStdout

	if err != nil {
		return "", errors.New("failed to convert to ts: " + err.Error())
	}

	tsSplit := strings.Split(ts, "export interface ")
	lastType := tsSplit[len(tsSplit)-1]
	lastTypeName := strings.Split(lastType, " ")[0]
	newLastTypeName := lastTypeName
	if len(newLastTypeName) == 0 {
		newLastTypeName = name
	}

	if count, exists := (*prereqsMap)[newLastTypeName]; exists {
		(*prereqsMap)[newLastTypeName]++
		newLastTypeName += "_" + strconv.Itoa(count+1)
	} else {
		(*prereqsMap)[newLastTypeName] = 1
	}

	rejoined := strings.Join(tsSplit, "interface ")
	rejoined = strings.Replace(rejoined, "interface "+lastTypeName, "interface "+newLastTypeName, 1)

	*target = newLastTypeName
	return rejoined, nil
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

// isIllegalCharacter checks if a character is illegal for TypeScript variable names
func isIllegalCharacter(r rune) bool {
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
}

// convertToTSVariableName converts a string to a TypeScript-safe variable name
func convertToTSVariableName(input string) string {
	var builder strings.Builder

	for _, r := range input {
		if isIllegalCharacter(r) {
			builder.WriteRune('_')
		} else {
			builder.WriteRune(r)
		}
	}

	result := builder.String()

	// Replace multiple underscores with a single underscore
	re := regexp.MustCompile(`_+`)
	result = re.ReplaceAllString(result, "_")

	// Remove leading underscores and numbers
	reLeading := regexp.MustCompile(`^[_0-9]+`)
	result = reLeading.ReplaceAllString(result, "")

	// Ensure the variable name does not start with a digit
	if len(result) == 0 || unicode.IsDigit(rune(result[0])) {
		result = "_" + result
	}

	return result
}
