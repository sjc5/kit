package rpc

const extraCode = `export type QueryAPIRoute = Extract<(typeof routes)[number], { type: "query" }>;
export type QueryAPIKey = QueryAPIRoute["key"];
export type QueryAPIInput<T extends QueryAPIKey> = Extract<
	QueryAPIRoute,
	{ key: T }
>["phantomInputType"];
export type QueryAPIOutput<T extends QueryAPIKey> = Extract<
	QueryAPIRoute,
	{ key: T }
>["phantomOutputType"];
export type QueryAPIRoutes = {
	[K in QueryAPIKey]: Extract<QueryAPIRoute, { key: K }>;
};

export type MutationAPIRoute = Extract<(typeof routes)[number], { type: "mutation" }>;
export type MutationAPIKey = MutationAPIRoute["key"];
export type MutationAPIRoutes = {
	[K in MutationAPIKey]: Extract<MutationAPIRoute, { key: K }>;
};
export type MutationAPIInput<T extends MutationAPIKey> = Extract<
	MutationAPIRoute,
	{ key: T }
>["phantomInputType"];
export type MutationAPIOutput<T extends MutationAPIKey> = Extract<
	MutationAPIRoute,
	{ key: T }
>["phantomOutputType"];
`
