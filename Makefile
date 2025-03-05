test-quiet:
	@go test ./...

test-loud:
	@go test -v ./...

bumper: test-quiet
	@go run ./scripts/bumper

bench-matcher:
	@go test -bench=. ./pkg/matcher

bench-router:
	@go test -bench=. ./pkg/router
