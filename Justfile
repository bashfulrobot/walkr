# repo-walker dev recipes

# Build the repo-walker binary
build:
    go build -o repo-walker .

# Build the .repo-walker walkthrough and serve it locally, opening a browser
serve:
    go run . serve .repo-walker --open

# Run the test suite
test:
    go test ./...

# Vet the code and check formatting
lint:
    go vet ./...
    gofmt -l .

# Build the .repo-walker walkthrough to ./site for a quick look
demo:
    go run . build .repo-walker -o ./site
    echo "open ./site/index.html"
