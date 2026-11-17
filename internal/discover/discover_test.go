package discover

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type fakeCFN struct {
	outputs []types.Output
	err     error
}

func (f fakeCFN) DescribeStacks(_ context.Context, _ *cloudformation.DescribeStacksInput,
	_ ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &cloudformation.DescribeStacksOutput{Stacks: []types.Stack{{Outputs: f.outputs}}}, nil
}

func TestEndpoint(t *testing.T) {
	cfn := fakeCFN{outputs: []types.Output{
		{OutputKey: aws.String("AppName"), OutputValue: aws.String("shortener")},
		{OutputKey: aws.String(OutputKey), OutputValue: aws.String("https://abc.execute-api.ap-south-1.amazonaws.com/Prod")},
	}}
	got, err := Endpoint(context.Background(), cfn, "URLShortener")
	if err != nil || got != "https://abc.execute-api.ap-south-1.amazonaws.com/Prod" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestEndpointMissingOutput(t *testing.T) {
	cfn := fakeCFN{outputs: []types.Output{{OutputKey: aws.String("AppName"), OutputValue: aws.String("x")}}}
	if _, err := Endpoint(context.Background(), cfn, "URLShortener"); !errors.Is(err, ErrOutputMissing) {
		t.Fatalf("want ErrOutputMissing, got %v", err)
	}
}

func TestEndpointDescribeError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := Endpoint(context.Background(), fakeCFN{err: boom}, "x"); !errors.Is(err, boom) {
		t.Fatalf("want wrapped error, got %v", err)
	}
}
