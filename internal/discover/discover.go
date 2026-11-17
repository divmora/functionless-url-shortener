// Package discover resolves the management API endpoint from CloudFormation stack outputs.
package discover

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// OutputKey is the stack output holding the execute-api management endpoint.
const OutputKey = "ManagementApiEndpoint"

// StackDescriber is the subset of the CloudFormation API used here.
type StackDescriber interface {
	DescribeStacks(ctx context.Context, in *cloudformation.DescribeStacksInput,
		optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

// ErrOutputMissing means the stack exists but predates the ManagementApiEndpoint output.
var ErrOutputMissing = errors.New("stack has no " + OutputKey + " output; redeploy the stack with v0.4.0+ or pass --endpoint")

// Endpoint returns the ManagementApiEndpoint output of the named stack.
func Endpoint(ctx context.Context, cfn StackDescriber, stack string) (string, error) {
	out, err := cfn.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{StackName: &stack})
	if err != nil {
		return "", fmt.Errorf("describing stack %q: %w", stack, err)
	}
	if len(out.Stacks) == 0 {
		return "", fmt.Errorf("stack %q not found", stack)
	}
	for _, o := range out.Stacks[0].Outputs {
		if o.OutputKey != nil && *o.OutputKey == OutputKey && o.OutputValue != nil {
			return *o.OutputValue, nil
		}
	}
	return "", ErrOutputMissing
}
