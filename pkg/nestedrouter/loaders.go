package nestedrouter

import "github.com/sjc5/kit/pkg/tasks"

type LoaderTask = tasks.Task
type Loader[O any] func(*Ctx) (O, error)

type LoaderMarker interface {
	GetOutputZeroValue() any
}

func (Loader[O]) GetOutputZeroValue() any {
	var zero O
	return zero
}

func (router *Router) AllLoaders() map[string]tasks.Task {
	return router.loaders
}

type Pattern = string

type LoaderResult struct {
	Pattern Pattern
	*tasks.TaskResult
}

type LoadersResultsMap map[Pattern]*LoaderResult
type LoadersResultsSlice []*LoaderResult

const (
	mapArg   = "map"
	sliceArg = "slice"
)

func (c *Ctx) GetLoaderResultsMap() (LoadersResultsMap, bool) {
	_, resultsMap, ok := c.runLoaders(mapArg)
	return resultsMap, ok
}

func (c *Ctx) GetLoaderResultsSlice() (LoadersResultsSlice, bool) {
	resultsSlice, _, ok := c.runLoaders(sliceArg)
	return resultsSlice, ok
}

func (c *Ctx) runLoaders(returnType string) (LoadersResultsSlice, LoadersResultsMap, bool) {
	matches, ok := c.FindMatches()
	if !ok {
		return nil, nil, false
	}

	lastMatch := matches[len(matches)-1]

	c.Params = lastMatch.Params
	c.SplatValues = lastMatch.SplatValues

	runArgs := make([]*tasks.RunArg, 0, len(matches))
	for _, match := range matches {
		loader, ok := c.router.loaders[match.Pattern()]
		if !ok {
			continue
		}
		runArgs = append(runArgs, &tasks.RunArg{Task: loader, Input: c})
	}

	results, ok := c.tasksCtx.Run(runArgs...)

	if returnType == mapArg {
		resultsMap := make(map[Pattern]*LoaderResult, len(matches))

		for _, match := range matches {
			loader, exists := c.router.loaders[match.Pattern()]
			if !exists {
				resultsMap[match.Pattern()] = &LoaderResult{Pattern: match.Pattern()}
				continue
			}
			resultsMap[match.Pattern()] = &LoaderResult{Pattern: match.Pattern(), TaskResult: loader.GetTaskResult(results)}
		}

		return nil, resultsMap, ok
	} else if returnType == sliceArg {
		resultsSlice := make([]*LoaderResult, 0, len(matches))

		for _, match := range matches {
			loader, exists := c.router.loaders[match.Pattern()]
			if !exists {
				resultsSlice = append(resultsSlice, &LoaderResult{Pattern: match.Pattern()})
				continue
			}
			resultsSlice = append(resultsSlice, &LoaderResult{Pattern: match.Pattern(), TaskResult: loader.GetTaskResult(results)})
		}

		return resultsSlice, nil, ok
	}

	panic("invalid returnType")
}
