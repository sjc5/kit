package router

import "net/http"

type Route struct {
	Segments []string `json:"segments"`
}

type MatchPreConditionChecker = func(r *http.Request, route *Route) bool

type Router struct {
	routes                   map[string]*Route
	matchPreConditionChecker MatchPreConditionChecker
}

type Match struct {
	Pattern string
	Params  Params
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) SetMatchPreConditionChecker(checker MatchPreConditionChecker) {
	r.matchPreConditionChecker = checker
}

func (r *Router) AddRoute(pattern string) {
	if r.routes == nil {
		r.routes = make(map[string]*Route)
	}

	r.routes[pattern] = &Route{Segments: ParseSegments(pattern)}
}

func (router *Router) FindBestMatch(r *http.Request) (*Match, bool) {
	realSegments := ParseSegments(r.URL.Path)
	var bestMatch *Match
	bestScore := -1

	for pattern, route := range router.routes {
		if router.matchPreConditionChecker != nil && !router.matchPreConditionChecker(r, route) {
			continue
		}

		if results, ok := MatchCore(route.Segments, realSegments); ok && results.Score > bestScore {
			bestScore = results.Score
			if bestMatch == nil {
				bestMatch = new(Match)
			}
			bestMatch.Params = results.Params
			bestMatch.Pattern = pattern
		}
	}

	return bestMatch, bestMatch != nil
}
