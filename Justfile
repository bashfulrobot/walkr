# walkr dev recipes

# Build the walkr binary
build:
    go build -o walkr .

# Build the .walkr walkthrough and serve it locally, opening a browser
serve:
    go run . serve .walkr --open

# Run the test suite
test:
    go test ./...

# Vet the code and check formatting (fails when gofmt lists any file)
lint:
    go vet ./...
    test -z "$(gofmt -l .)" || { gofmt -l .; exit 1; }

# Full pre-merge gate: lint, tests, and a dogfood build of .walkr
check: lint test
    out="$(mktemp -d)" && go run . build .walkr -o "$out" && test -f "$out/index.html" && rm -rf "$out"

# Build the .walkr walkthrough to ./site for a quick look
demo:
    go run . build .walkr -o ./site
    echo "open ./site/index.html"
