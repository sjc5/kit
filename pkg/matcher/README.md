- segments starting with double-underscores (`__whatever`) are ignored
- segments starting with a dollar sign (`$whatever`) signify a dynamic value
- segments equal to `_index` (must be last segment) signify an index route

// **TODO test bad pattern strings -- or require pre-validated patterns ?
// **TODO can we extract out a normal (non-nested) router for basic APIs? then build this on top of it?
// \_\_TODO should this be structured as an instance with methods and an internal cache? or maybe a small wrapper on top of this that does that?
