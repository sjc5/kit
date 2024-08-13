package rpc

import (
	"errors"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
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

type nameAndDef struct {
	Name string
	Def  string
}

func generateTypeScriptContent(opts Opts) (string, error) {
	prereqsMap := make(map[string]int)
	seenTypes := make(seenTypes)
	prereqs := make([]nameAndDef, 0)
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

			locPrereqs, err := makeTSStr(makeTSStrInput{
				target:         &inputStr,
				t:              routeDef.Input,
				prereqsMap:     &prereqsMap,
				name:           convertToTSVariableName(namePrefix + "_input"),
				seenTypes:      &seenTypes,
				nameIsOverride: true,
			})
			if err != nil {
				return "", errors.New("failed to convert input to ts: " + err.Error())
			}

			prereqs = append(prereqs, locPrereqs...)
		} else {
			inputStr = "undefined"
		}

		if routeDef.Output != nil {
			namePrefix := routeDef.Key
			if namePrefix == "" {
				namePrefix = "AnonType"
			}

			locPrereqs, err := makeTSStr(makeTSStrInput{
				target:         &outputStr,
				t:              routeDef.Output,
				prereqsMap:     &prereqsMap,
				name:           convertToTSVariableName(namePrefix + "_output"),
				seenTypes:      &seenTypes,
				nameIsOverride: true,
			})
			if err != nil {
				return "", errors.New("failed to convert output to ts: " + err.Error())
			}

			prereqs = append(prereqs, locPrereqs...)
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
		ts = intro + nameAndDefListToTsStr(prereqs) + ts
	} else {
		ts = intro
	}

	// NOW HANDLE AD HOC TYPES AT END
	// We want to use the same prereqs map, but wipe the
	// old prereqs because they're already concatenated
	if len(opts.AdHocTypes) > 0 {
		prereqs = make([]nameAndDef, 0)

		for _, adHocType := range opts.AdHocTypes {
			target := ""
			name := convertToTSVariableName(adHocType.Name)

			locPrereqs, err := makeTSStr(makeTSStrInput{
				target:         &target,
				t:              adHocType.Struct,
				prereqsMap:     &prereqsMap,
				name:           name,
				seenTypes:      &seenTypes,
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

func getIsAnonName(name string) bool {
	return len(name) == 0 || name == " " || name == "_"
}

type makeTSStrInput struct {
	target         *string
	t              any
	prereqsMap     *map[string]int
	name           string
	seenTypes      *seenTypes
	nameIsOverride bool
}

func makeTSStr(input makeTSStrInput) ([]nameAndDef, error) {
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
		return nil, errors.New("failed to convert to ts: " + err.Error())
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
				newCurrentName = "__AnonType"
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
			newCurrentName += "_" + strconv.Itoa(count+1)
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
			Name: tsSplitNames[i],
			Def:  ts,
		}
	}

	*input.target = newFinalTypeName
	return nameAndDefList, nil
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

func nameAndDefListToTsStr(nameAndDefList []nameAndDef) string {
	ts := ""
	for _, item := range nameAndDefList {
		if item.Name == "" || item.Def == "" {
			continue
		}
		ts += "export interface " + item.Name + " " + item.Def + "\n"
	}
	return ts
}
