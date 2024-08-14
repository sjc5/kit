package rpc

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type TypeDefFields = map[string]string

type RouteDef struct {
	Key    string
	Type   Type
	Input  any
	Output any
}

type Type string

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

// seenTypes is a map of trimmed, sans-name type definition strings to a slice of used names
type seenTypes = map[trimmedType][]cleanName
type trimmedType = string
type cleanName = string

func GenerateTypeScript(opts Opts) error {
	ts, err := generateTypeScriptContent(opts)
	if err != nil {
		return err
	}
	return writeTSFile(opts.OutDest, ts)
}

type innerTypeDef struct {
	fields              TypeDefFields
	inputFinalTypeName  string
	outputFinalTypeName string
}

type nameAndDef struct {
	name string
	def  string
}

func processRouteDef(routeDef RouteDef, prereqsMap *map[string]int, seenTypes *seenTypes) (*innerTypeDef, []nameAndDef, error) {
	inputFinalTypeName, inputPrereqs, err := processRouteT(
		baseInput{t: routeDef.Input, prereqsMap: prereqsMap, seenTypes: seenTypes},
		convertToPascalCase(routeDef.Key+"Input"),
	)
	if err != nil {
		return nil, nil, err
	}

	outputFinalTypeName, outputPrereqs, err := processRouteT(
		baseInput{t: routeDef.Output, prereqsMap: prereqsMap, seenTypes: seenTypes},
		convertToPascalCase(routeDef.Key+"Output"),
	)
	if err != nil {
		return nil, nil, err
	}

	itd := &innerTypeDef{
		fields:              map[string]string{"key": routeDef.Key, "type": string(routeDef.Type)},
		inputFinalTypeName:  inputFinalTypeName,
		outputFinalTypeName: outputFinalTypeName,
	}
	combinedPrereqs := append(inputPrereqs, outputPrereqs...)

	return itd, combinedPrereqs, nil
}

type baseInput struct {
	t          any
	prereqsMap *map[string]int
	seenTypes  *seenTypes
}

func processRouteT(input baseInput, name string) (string, []nameAndDef, error) {
	if input.t == nil {
		return "undefined", nil, nil
	}

	finalTypeName, prereqs, err := makeTSStr(makeTSStrInput{
		baseInput:      input,
		name:           name,
		nameIsOverride: true,
	})
	if err != nil {
		return "", nil, errors.New("failed to convert to ts: " + err.Error())
	}

	return finalTypeName, prereqs, nil
}

// generateTypeScriptContent generates TypeScript content from the given options
func generateTypeScriptContent(opts Opts) (string, error) {
	prereqsMap := make(map[string]int)
	seenTypes := make(seenTypes)
	prereqs := make([]nameAndDef, 0)
	routeTS := "\nconst routes = ["

	for _, routeDef := range opts.RouteDefs {
		itd, locPrereqs, err := processRouteDef(routeDef, &prereqsMap, &seenTypes)
		if err != nil {
			return "", err
		}
		prereqs = append(prereqs, locPrereqs...)
		routeTS += itdToStr(*itd)
	}

	ts := "/*\n * This file is auto-generated. Do not edit.\n */\n"

	if len(opts.RouteDefs) > 0 {
		ts += nameAndDefListToTsStr(prereqs) + routeTS + "\n] as const;\n\n" + extraCode
	}

	// NOW HANDLE AD HOC TYPES AT END
	// We want to use the same prereqs map, but wipe the
	// old prereqs because they're already concatenated
	if len(opts.AdHocTypes) > 0 {
		prereqs = make([]nameAndDef, 0)

		for _, adHocType := range opts.AdHocTypes {
			name := convertToPascalCase(adHocType.Name)

			_, locPrereqs, err := makeTSStr(makeTSStrInput{
				baseInput: baseInput{
					t:          adHocType.Struct,
					prereqsMap: &prereqsMap,
					seenTypes:  &seenTypes,
				},
				name:           name,
				nameIsOverride: !getIsAnonName(name),
			})
			if err != nil {
				return "", errors.New("failed to convert ad hoc type to ts: " + err.Error())
			}

			prereqs = append(prereqs, locPrereqs...)
		}

		ts += "\n/*\n * AD HOC TYPES (skipped if already exported above)\n */\n"
		ts += nameAndDefListToTsStr(prereqs)
	}

	return ts, nil
}

type makeTSStrInput struct {
	baseInput
	name           string
	nameIsOverride bool
}

func makeTSStr(input makeTSStrInput) (string, []nameAndDef, error) {
	converter := newConverter()
	converter.Add(input.t)

	// quiet typescriptify logs
	oldStdout := os.Stdout
	null, _ := os.Open(os.DevNull)
	os.Stdout = null
	ts, err := converter.Convert(make(map[string]string))
	null.Close()
	os.Stdout = oldStdout

	if err != nil {
		return "", nil, errors.New("failed to convert to ts: " + err.Error())
	}

	tsSplit := []string{}
	for _, ts := range strings.Split(ts, "export interface ") {
		trimmed := strings.TrimSpace(ts)
		if trimmed != "" {
			tsSplit = append(tsSplit, ts)
		}
	}

	newFinalTypeName := ""

	tsSplitNames := make([]string, len(tsSplit))

	for i, currentType := range tsSplit {
		if strings.HasPrefix(currentType, " {") {
			currentType = input.name + currentType
			tsSplit[i] = currentType
		}

		currentName := strings.Split(currentType, " ")[0]
		newCurrentName := currentName
		isLastAndOverride := i == len(tsSplit)-1 && input.nameIsOverride
		if getIsAnonName(newCurrentName) || isLastAndOverride {
			newCurrentName = input.name
			if getIsAnonName(newCurrentName) {
				newCurrentName = "AnonType"
			}
		}

		trimmed := strings.TrimSpace(currentType)
		trimmed = "{" + strings.Split(trimmed, " {")[1]

		if usedNames, typeWithoutNameAlreadySeen := (*input.seenTypes)[trimmed]; typeWithoutNameAlreadySeen {
			isAnonDef := strings.HasPrefix(currentType, " {")
			if !isAnonDef && slices.Contains(usedNames, newCurrentName) {
				tsSplit[i] = ""
				continue
			} else if !isAnonDef {
				(*input.seenTypes)[trimmed] = append(usedNames, newCurrentName)
			}
		} else {
			(*input.seenTypes)[trimmed] = append((*input.seenTypes)[trimmed], newCurrentName)
		}

		if count, exists := (*input.prereqsMap)[newCurrentName]; exists {
			(*input.prereqsMap)[newCurrentName]++
			newCurrentName += strconv.Itoa(count + 1)
		} else {
			(*input.prereqsMap)[newCurrentName] = 1
		}

		if i == len(tsSplit)-1 {
			newFinalTypeName = newCurrentName
		}

		tsSplit[i] = strings.TrimSpace(strings.Replace(currentType, currentName+" {", " {", 1))
		tsSplitNames[i] = strings.TrimSpace(newCurrentName)
	}

	nameAndDefList := make([]nameAndDef, len(tsSplit))

	for i, ts := range tsSplit {
		if ts == "" || tsSplitNames[i] == "" {
			continue
		}
		nameAndDefList[i] = nameAndDef{
			name: tsSplitNames[i],
			def:  ts,
		}
	}

	return newFinalTypeName, nameAndDefList, nil
}

// nameAndDefListToTsStr converts a list of nameAndDef to a TypeScript string
func nameAndDefListToTsStr(nameAndDefList []nameAndDef) string {
	ts := ""
	for _, item := range nameAndDefList {
		if item.name == "" || item.def == "" {
			continue
		}
		ts += "export type " + item.name + " = " + item.def + "\n"
	}
	return ts
}

// getIsAnonName checks if a name is anonymous
func getIsAnonName(name string) bool {
	return len(name) == 0 || name == " " || name == "_"
}

func itdToStr(itd innerTypeDef) string {
	var builder strings.Builder

	builder.WriteString("\n\t{\n")

	keys := make([]string, 0, len(itd.fields))
	for k := range itd.fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("\t\t%s: \"%s\",\n", key, itd.fields[key]))
	}

	builder.WriteString(fmt.Sprintf("\t\tphantomInputType: null as unknown as %s,\n", itd.inputFinalTypeName))
	builder.WriteString(fmt.Sprintf("\t\tphantomOutputType: null as unknown as %s,\n", itd.outputFinalTypeName))

	builder.WriteString("\t},")

	return builder.String()
}
