package rpc

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

type trimmedType = string
type cleanName = string

// "seenTypes" is a map of trimmed, sans-name type definition strings to a slice of used names

func GenerateTypeScript(opts Opts) error {
	err := fsutil.EnsureDir(opts.OutDest)
	if err != nil {
		return errors.New("failed to ensure out dest dir: " + err.Error())
	}
	prereqsMap := make(map[string]int)
	seenTypes := make(map[trimmedType][]cleanName)
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
			locPrereqs, err := makeTSStr(&inputStr, routeDef.Input, &prereqsMap, name, &seenTypes, true)
			if err != nil {
				return errors.New("failed to convert input to ts: " + err.Error())
			}
			prereqs += locPrereqs
		} else {
			inputStr = "undefined"
		}

		if routeDef.Output != nil {
			namePrefix := routeDef.Key
			if namePrefix == "" {
				namePrefix = "AnonType"
			}
			name := convertToTSVariableName(namePrefix + "_output")
			locPrereqs, err := makeTSStr(&outputStr, routeDef.Output, &prereqsMap, name, &seenTypes, true)
			if err != nil {
				return errors.New("failed to convert output to ts: " + err.Error())
			}
			prereqs += locPrereqs
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
			locPrereqs, err := makeTSStr(&target, adHocType.Struct, &prereqsMap, name, &seenTypes, !getIsAnonName(name))
			if err != nil {
				return errors.New("failed to convert ad hoc type to ts: " + err.Error())
			}
			prereqs += locPrereqs
		}

		ts += "\n/*\n * AD HOC TYPES (skipped if already exported above)\n */\n" + prereqs
	}

	// Now write to disk
	err = os.WriteFile(filepath.Join(opts.OutDest, "api-types.ts"), []byte(ts), os.ModePerm)
	if err != nil {
		return errors.New("failed to write ts file: " + err.Error())
	}

	return nil
}

func getIsAnonName(name string) bool {
	return len(name) == 0 || name == " " || name == "_"
}

func makeTSStr(
	target *string,
	t any,
	prereqsMap *map[string]int,
	name string,
	seenTypes *map[trimmedType][]cleanName,
	nameIsOverride bool,
) (string, error) {
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

	rejoinStr := "export interface "

	tsSplit := []string{}
	for _, ts := range strings.Split(ts, rejoinStr) {
		trimmed := strings.TrimSpace(ts)
		if trimmed != "" {
			tsSplit = append(tsSplit, ts)
		}
	}

	newFinalTypeName := ""

	for i, currentType := range tsSplit {
		if strings.HasPrefix(currentType, " {") {
			currentType = name + currentType
			tsSplit[i] = currentType
		}

		currentName := strings.Split(currentType, " ")[0]
		newCurrentName := currentName
		isLastAndOverride := i == len(tsSplit)-1 && nameIsOverride
		if getIsAnonName(newCurrentName) || isLastAndOverride {
			newCurrentName = name
			if getIsAnonName(newCurrentName) {
				newCurrentName = "__AnonType"
			}
		}

		trimmed := strings.TrimSpace(currentType)
		trimmed = "{" + strings.Split(trimmed, " {")[1]

		if usedNames, typeWithoutNameAlreadySeen := (*seenTypes)[trimmed]; typeWithoutNameAlreadySeen {
			isAnonDef := strings.HasPrefix(currentType, " {")
			if !isAnonDef && slices.Contains(usedNames, newCurrentName) {
				tsSplit[i] = ""
				continue
			} else if !isAnonDef {
				(*seenTypes)[trimmed] = append(usedNames, newCurrentName)
			}
		} else {
			(*seenTypes)[trimmed] = append((*seenTypes)[trimmed], newCurrentName)
		}

		if count, exists := (*prereqsMap)[newCurrentName]; exists {
			(*prereqsMap)[newCurrentName]++
			newCurrentName += "_" + strconv.Itoa(count+1)
		} else {
			(*prereqsMap)[newCurrentName] = 1
		}

		if i == len(tsSplit)-1 {
			newFinalTypeName = newCurrentName
		}

		tsSplit[i] = strings.Replace(currentType, currentName+" {", newCurrentName+" {", 1)
	}

	cleanedTsSplit := []string{""} // start with empty string to keep the first export interface
	for _, ts := range tsSplit {
		trimmed := strings.TrimSpace(ts)
		if trimmed != "" {
			cleanedTsSplit = append(cleanedTsSplit, ts)
		}
	}

	rejoined := strings.Join(cleanedTsSplit, rejoinStr)
	cleanedRejoined := strings.TrimSpace(rejoined)
	if cleanedRejoined != "" {
		cleanedRejoined += "\n"
	}
	*target = newFinalTypeName
	return cleanedRejoined, nil
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
