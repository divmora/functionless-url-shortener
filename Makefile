.PHONY: all validate build deploy deploy-guided clean

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

clean:
	@echo "Cleaning build artifacts..."
	rm -rf .aws-sam/
