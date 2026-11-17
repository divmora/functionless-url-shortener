.PHONY: all validate build deploy deploy-guided test cli clean

all: validate build

validate:
	@echo "Validating SAM template..."
	sam validate -t template.yaml
	@which cfn-lint > /dev/null 2>&1 && cfn-lint template.yaml || echo "cfn-lint not installed (optional)"

build:
	@echo "Building SAM application..."
	sam build

deploy:
	@echo "Deploying SAM stack..."
	sam deploy

deploy-guided:
	@echo "Starting guided deployment..."
	sam deploy -g

test:
	@echo "Running Go tests..."
	go test ./...

cli:
	@echo "Building shorten CLI..."
	go build -o bin/shorten ./cmd/shorten

clean:
	@echo "Cleaning build artifacts..."
	rm -rf .aws-sam/ bin/
