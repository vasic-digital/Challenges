package userflow

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Compile-time interface check.
var _ GRPCAdapter = (*GRPCCLIAdapter)(nil)

// GRPCAdapter defines the interface for gRPC service testing.
// Implementations wrap gRPC tooling (e.g., grpcurl) to perform
// service discovery, method invocation, and health checking.
type GRPCAdapter interface {
	// ListServices returns all available gRPC services via
	// server reflection.
	ListServices(ctx context.Context) ([]string, error)

	// ListMethods returns all methods for the given service.
	ListMethods(
		ctx context.Context, service string,
	) ([]string, error)

	// Invoke calls a gRPC method with a JSON request body
	// and returns the JSON response.
	Invoke(
		ctx context.Context, method, request string,
	) (string, error)

	// InvokeStream calls a server-streaming gRPC method and
	// returns all collected JSON responses.
	InvokeStream(
		ctx context.Context, method, request string,
	) ([]string, error)

	// HealthCheck performs a gRPC health check for the given
	// service name. Pass empty string for the overall server.
	HealthCheck(
		ctx context.Context, service string,
	) (bool, error)

	// Available returns true if the gRPC server is reachable.
	Available(ctx context.Context) bool

	// Close releases any resources held by the adapter.
	Close(ctx context.Context) error
}

// GRPCOption configures a GRPCCLIAdapter.
type GRPCOption func(*grpcCLIConfig)

// grpcCLIConfig holds configuration for the grpcurl CLI
// adapter.
type grpcCLIConfig struct {
	tls        bool
	insecure   bool
	certFile   string
	protoFiles []string
	headers    map[string]string
}

// WithTLS enables TLS for gRPC connections.
func WithTLS() GRPCOption {
	return func(c *grpcCLIConfig) {
		c.tls = true
	}
}

// WithCert sets the TLS certificate file path.
func WithCert(certFile string) GRPCOption {
	return func(c *grpcCLIConfig) {
		c.certFile = certFile
		c.tls = true
	}
}

// WithInsecure disables TLS certificate verification.
func WithInsecure() GRPCOption {
	return func(c *grpcCLIConfig) {
		c.insecure = true
	}
}

// WithProtoFiles sets proto file paths for services that do
// not support server reflection.
func WithProtoFiles(files ...string) GRPCOption {
	return func(c *grpcCLIConfig) {
		c.protoFiles = append(c.protoFiles, files...)
	}
}

// WithHeaders sets additional metadata headers to send with
// each gRPC request.
func WithHeaders(headers map[string]string) GRPCOption {
	return func(c *grpcCLIConfig) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// GRPCCLIAdapter implements GRPCAdapter using the grpcurl
// command-line tool. It shells out to grpcurl for each
// operation, parsing stdout as the response.
type GRPCCLIAdapter struct {
	serverAddr string
	config     grpcCLIConfig
}

// NewGRPCCLIAdapter creates a GRPCCLIAdapter targeting the
// given server address (host:port) with optional configuration.
func NewGRPCCLIAdapter(
	serverAddr string, opts ...GRPCOption,
) *GRPCCLIAdapter {
	cfg := grpcCLIConfig{
		headers: make(map[string]string),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &GRPCCLIAdapter{
		serverAddr: serverAddr,
		config:     cfg,
	}
}

// ListServices returns all gRPC services advertised via
// server reflection.
func (a *GRPCCLIAdapter) ListServices(
	ctx context.Context,
) ([]string, error) {
	args := a.baseArgs()
	args = append(args, a.serverAddr, "list")

	out, err := a.runGRPCurl(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	return parseLines(out), nil
}

// ListMethods returns all methods for the given service.
func (a *GRPCCLIAdapter) ListMethods(
	ctx context.Context, service string,
) ([]string, error) {
	args := a.baseArgs()
	args = append(args, a.serverAddr, "list", service)

	out, err := a.runGRPCurl(ctx, args)
	if err != nil {
		return nil, fmt.Errorf(
			"list methods for %s: %w", service, err,
		)
	}

	return parseLines(out), nil
}

// Invoke calls a unary gRPC method with a JSON request body
// and returns the JSON response string.
func (a *GRPCCLIAdapter) Invoke(
	ctx context.Context, method, request string,
) (string, error) {
	args := a.baseArgs()
	if request != "" {
		args = append(args, "-d", request)
	}
	args = append(args, a.serverAddr, method)

	out, err := a.runGRPCurl(ctx, args)
	if err != nil {
		return "", fmt.Errorf(
			"invoke %s: %w", method, err,
		)
	}

	return strings.TrimSpace(out), nil
}

// InvokeStream calls a server-streaming gRPC method and
// returns all JSON response objects collected from stdout.
// Each response object is separated by newlines in the
// grpcurl output.
func (a *GRPCCLIAdapter) InvokeStream(
	ctx context.Context, method, request string,
) ([]string, error) {
	args := a.baseArgs()
	if request != "" {
		args = append(args, "-d", request)
	}
	args = append(args, a.serverAddr, method)

	out, err := a.runGRPCurl(ctx, args)
	if err != nil {
		return nil, fmt.Errorf(
			"invoke stream %s: %w", method, err,
		)
	}

	return parseJSONObjects(out), nil
}

// HealthCheck performs a gRPC health check using the standard
// grpc.health.v1.Health/Check method.
func (a *GRPCCLIAdapter) HealthCheck(
	ctx context.Context, service string,
) (bool, error) {
	request := "{}"
	if service != "" {
		request = fmt.Sprintf(
			`{"service":"%s"}`, service,
		)
	}

	args := a.baseArgs()
	args = append(args, "-d", request)
	args = append(
		args,
		a.serverAddr,
		"grpc.health.v1.Health/Check",
	)

	out, err := a.runGRPCurl(ctx, args)
	if err != nil {
		return false, fmt.Errorf(
			"health check: %w", err,
		)
	}

	return strings.Contains(out, "SERVING"), nil
}

// Available returns true if the gRPC server is reachable by
// attempting to list services.
func (a *GRPCCLIAdapter) Available(
	ctx context.Context,
) bool {
	_, err := a.ListServices(ctx)
	return err == nil
}

// Close is a no-op for the CLI adapter since grpcurl does not
// maintain persistent connections.
func (a *GRPCCLIAdapter) Close(
	_ context.Context,
) error {
	return nil
}

// baseArgs constructs the common grpcurl arguments from the
// adapter configuration.
func (a *GRPCCLIAdapter) baseArgs() []string {
	var args []string

	if !a.config.tls {
		args = append(args, "-plaintext")
	}
	if a.config.insecure {
		args = append(args, "-insecure")
	}
	if a.config.certFile != "" {
		args = append(args, "-cacert", a.config.certFile)
	}
	for _, pf := range a.config.protoFiles {
		args = append(args, "-proto", pf)
	}
	for k, v := range a.config.headers {
		args = append(
			args, "-H", fmt.Sprintf("%s: %s", k, v),
		)
	}

	return args
}

// runGRPCurl executes a grpcurl command with the given
// arguments and returns stdout. If the command fails, the
// error includes stderr content.
func (a *GRPCCLIAdapter) runGRPCurl(
	ctx context.Context, args []string,
) (string, error) {
	cmd := exec.CommandContext(ctx, "grpcurl", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf(
			"grpcurl %s: %s: %w",
			strings.Join(args, " "), errMsg, err,
		)
	}

	return stdout.String(), nil
}

// parseLines splits output into non-empty trimmed lines.
func parseLines(output string) []string {
	raw := strings.Split(output, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

// parseJSONObjects splits grpcurl streaming output into
// individual JSON objects. grpcurl separates streamed
// responses with blank lines or outputs them as separate
// JSON blocks delimited by braces.
func parseJSONObjects(output string) []string {
	var objects []string
	var current strings.Builder
	depth := 0

	for _, ch := range output {
		switch ch {
		case '{':
			depth++
			current.WriteRune(ch)
		case '}':
			depth--
			current.WriteRune(ch)
			if depth == 0 {
				obj := strings.TrimSpace(current.String())
				if obj != "" {
					objects = append(objects, obj)
				}
				current.Reset()
			}
		default:
			if depth > 0 {
				current.WriteRune(ch)
			}
		}
	}

	return objects
}
