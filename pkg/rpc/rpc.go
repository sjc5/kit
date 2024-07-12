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

type AdHocType struct {
	// Instance of the struct to generate TypeScript for
	Struct any

	// Name is required only if struct is anonymous, otherwise optional override
	Name string
}

type Opts struct {
	OutDest    string
	RouteDefs  []RouteDef
	AdHocTypes []AdHocType
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
			locPrereqs, err := makeTSStr(&inputStr, routeDef.Input, &prereqsMap, name, false)
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
			locPrereqs, err := makeTSStr(&outputStr, routeDef.Output, &prereqsMap, name, false)
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

	intro := "/*\n * This file is auto-generated. Do not edit.\n */\n"
	var ts string

	if len(opts.RouteDefs) > 0 {
		tsEnd := "\n] as const;"
		queryTS += tsEnd
		mutationTS += tsEnd
		ts = ts + queryTS + "\n" + mutationTS + "\n"

		ts += "\n" + extraCode
		ts = intro + prereqs + ts
	} else {
		ts = intro
	}

	// NOW HANDLE AD HOC TYPES AT END
	// We want to use the same prereqs map, but wipe the
	// old prereqs because they're already concatenated
	if len(opts.AdHocTypes) > 0 {
		prereqs = ""

		for _, adHocType := range opts.AdHocTypes {
			target := ""
			name := convertToTSVariableName(adHocType.Name)
			locPrereqs, err := makeTSStr(&target, adHocType.Struct, &prereqsMap, name, true)
			if err != nil {
				return errors.New("failed to convert ad hoc type to ts: " + err.Error())
			}
			if locPrereqs != "" {
				prereqs += locPrereqs
			}
		}

		ts += "\n\n/*\n * AD HOC TYPES\n */\n" + prereqs
	}

	// Now write to disk
	err = os.WriteFile(filepath.Join(opts.OutDest, "api-types.ts"), []byte(ts), os.ModePerm)
	if err != nil {
		return errors.New("failed to write ts file: " + err.Error())
	}

	return nil
}

func makeTSStr(target *string, t any, prereqsMap *map[string]int, name string, keepExport bool) (string, error) {
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

	if strings.HasPrefix(lastType, " {") {
		lastType = name + lastType
		tsSplit[len(tsSplit)-1] = lastType
	}

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

	rejoinStr := "interface "
	if keepExport {
		rejoinStr = "export interface "
	}
	rejoined := strings.Join(tsSplit, rejoinStr)
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
