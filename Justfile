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

# Vet the code and check formatting
lint:
    go vet ./...
    gofmt -l .

# Build the .walkr walkthrough to ./site for a quick look
demo:
    go run . build .walkr -o ./site
    echo "open ./site/index.html"
