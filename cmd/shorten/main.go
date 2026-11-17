// Command shorten manages links on a Functionless URL Shortener deployment.
//
// It signs requests with SigV4 using the standard AWS credential chain
// (environment, shared config/credentials, IAM Identity Center, instance roles).
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"

	"github.com/divmora/functionless-url-shortener/internal/client"
	"github.com/divmora/functionless-url-shortener/internal/discover"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	exitOK       = 0
	exitError    = 1
	exitUsage    = 2
	exitNotFound = 3
	exitExists   = 4
)

const usage = `shorten - manage links on a Functionless URL Shortener deployment

Usage:
  shorten create <url> [--id <slug>] [--ttl <seconds>]
  shorten get <id>
  shorten delete <id>
  shorten version

Connection flags (any command):
  --stack <name>      CloudFormation stack to read the endpoint from (env SHORTEN_STACK, default "URLShortener")
  --endpoint <url>    Management API endpoint, skips stack lookup (env SHORTEN_ENDPOINT)
  --region <region>   AWS region (defaults to the AWS config / endpoint host)
  --profile <name>    AWS shared config profile
  --output text|json  Output format (default text)

Exit codes: 0 ok, 1 error, 2 usage, 3 not found, 4 ID already exists.
`

type globalFlags struct {
	stack, endpoint, region, profile, output string
}

func (g *globalFlags) register(fs *flag.FlagSet) {
	fs.StringVar(&g.stack, "stack", envOr("SHORTEN_STACK", "URLShortener"), "")
	fs.StringVar(&g.endpoint, "endpoint", os.Getenv("SHORTEN_ENDPOINT"), "")
	fs.StringVar(&g.region, "region", "", "")
	fs.StringVar(&g.profile, "profile", "", "")
	fs.StringVar(&g.output, "output", "text", "")
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		_, _ = fmt.Fprint(stdout, usage)
		if len(args) == 0 {
			return exitUsage
		}
		return exitOK
	}
	cmd, rest := args[0], args[1:]
	if cmd == "version" {
		_, _ = fmt.Fprintln(stdout, version)
		return exitOK
	}

	var g globalFlags
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	g.register(fs)
	var id string
	var ttl int64
	if cmd == "create" {
		fs.StringVar(&id, "id", "", "")
		fs.Int64Var(&ttl, "ttl", 0, "")
	}
	positional, err := parseInterspersed(fs, rest)
	if err != nil {
		return usageErr(stderr, err.Error())
	}
	if g.output != "text" && g.output != "json" {
		return usageErr(stderr, "--output must be text or json")
	}

	switch cmd {
	case "create":
		if len(positional) != 1 {
			return usageErr(stderr, "create takes exactly one <url>")
		}
		if err := validateURL(positional[0]); err != nil {
			return usageErr(stderr, err.Error())
		}
		if ttl < 0 {
			return usageErr(stderr, "--ttl must be a positive number of seconds")
		}
		if id == "" {
			if id, err = randomID(7); err != nil {
				return fail(stderr, err)
			}
		} else if err := validateID(id); err != nil {
			return usageErr(stderr, err.Error())
		}
	case "get", "delete":
		if len(positional) != 1 {
			return usageErr(stderr, cmd+" takes exactly one <id>")
		}
		id = positional[0]
		if err := validateID(id); err != nil {
			return usageErr(stderr, err.Error())
		}
	default:
		return usageErr(stderr, fmt.Sprintf("unknown command %q", cmd))
	}

	c, err := newClient(ctx, g)
	if err != nil {
		return fail(stderr, err)
	}

	var link *client.Link
	switch cmd {
	case "create":
		link, err = c.Create(ctx, client.CreateInput{ID: id, URL: positional[0], TTL: ttl})
	case "get":
		link, err = c.Get(ctx, id)
	case "delete":
		link, err = c.Delete(ctx, id)
	}
	if err != nil {
		return fail(stderr, err)
	}
	if link.ID == "" {
		link.ID = id
	}
	printLink(stdout, g.output, link)
	if cmd == "delete" && g.output == "text" {
		_, _ = fmt.Fprintln(stderr, "note: CloudFront may keep serving the cached redirect for up to 5 minutes.")
	}
	return exitOK
}

// parseInterspersed allows flags before or after positional arguments.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}

func newClient(ctx context.Context, g globalFlags) (*client.Client, error) {
	var opts []func(*config.LoadOptions) error
	if g.profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(g.profile))
	}
	region := g.region
	if region == "" && g.endpoint != "" {
		region = regionFromEndpoint(g.endpoint)
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	if cfg.Region == "" {
		return nil, errors.New("no AWS region set; pass --region or set AWS_REGION")
	}

	endpoint := g.endpoint
	if endpoint == "" {
		endpoint, err = discover.Endpoint(ctx, cloudformation.NewFromConfig(cfg), g.stack)
		if err != nil {
			return nil, err
		}
	}
	return client.New(endpoint, cfg.Region, aws.NewCredentialsCache(cfg.Credentials)), nil
}

var executeAPIHost = regexp.MustCompile(`^[a-z0-9]+\.execute-api\.([a-z0-9-]+)\.amazonaws\.com$`)

func regionFromEndpoint(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	if m := executeAPIHost.FindStringSubmatch(u.Hostname()); m != nil {
		return m[1]
	}
	return ""
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func validateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid id %q: use 1-128 letters, digits, '-' or '_'", id)
	}
	return nil
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("invalid url %q: must be an absolute http(s) URL", raw)
	}
	return nil
}

const idAlphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// randomID returns an unambiguous base57 slug (no 0/O/1/l/I).
func randomID(n int) (string, error) {
	var b strings.Builder
	max := big.NewInt(int64(len(idAlphabet)))
	for range n {
		i, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b.WriteByte(idAlphabet[i.Int64()])
	}
	return b.String(), nil
}

func printLink(w io.Writer, format string, l *client.Link) {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(l)
		return
	}
	rows := [][2]string{{"id", l.ID}, {"url", l.URL}, {"shortUrl", l.ShortURL}, {"owner", l.Owner}, {"timestamp", l.Timestamp}}
	for _, r := range rows {
		if r[1] != "" {
			_, _ = fmt.Fprintf(w, "%-10s %s\n", r[0]+":", r[1])
		}
	}
}

func usageErr(w io.Writer, msg string) int {
	_, _ = fmt.Fprintf(w, "error: %s\n\nRun 'shorten --help' for usage.\n", msg)
	return exitUsage
}

func fail(w io.Writer, err error) int {
	_, _ = fmt.Fprintf(w, "error: %v\n", err)
	switch {
	case errors.Is(err, client.ErrNotFound):
		return exitNotFound
	case errors.Is(err, client.ErrExists):
		return exitExists
	case errors.Is(err, client.ErrForbidden):
		_, _ = fmt.Fprintln(w, "hint: the caller must belong to the stack's AWS Organization and be allowed execute-api:Invoke on /api.")
	}
	return exitError
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
