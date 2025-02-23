package router

var Version = struct {
	LinearMatcherCore string
	TrieV1            string
	TrieV2            string
}{
	LinearMatcherCore: "LinearMatcherCore",
	TrieV1:            "TrieV1",
	TrieV2:            "TrieV2",
}

var VERSION = Version.LinearMatcherCore
