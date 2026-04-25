package userflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ GRPCAdapter = (*GRPCCLIAdapter)(nil)

func TestNewGRPCCLIAdapter(t *testing.T) {
	tests := []struct {
		name       string
		serverAddr string
		opts       []GRPCOption
		wantAddr   string
		wantTLS    bool
	}{
		{
			name:       "basic_address",
			serverAddr: "localhost:50051",
			wantAddr:   "localhost:50051",
		},
		{
			name:       "empty_address",
			serverAddr: "",
			wantAddr:   "",
		},
		{
			name:       "remote_address",
			serverAddr: "grpc.example.com:443",
			wantAddr:   "grpc.example.com:443",
		},
		{
			name:       "with_tls",
			serverAddr: "localhost:50051",
			opts:       []GRPCOption{WithTLS()},
			wantAddr:   "localhost:50051",
			wantTLS:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewGRPCCLIAdapter(
				tt.serverAddr, tt.opts...,
			)
			require.NotNil(t, adapter)
			assert.Equal(
				t, tt.wantAddr, adapter.serverAddr,
			)
			assert.Equal(
				t, tt.wantTLS, adapter.config.tls,
			)
			assert.NotNil(t, adapter.config.headers)
		})
	}
}

func TestGRPCCLIAdapter_Options(t *testing.T) {
	t.Run("WithTLS", func(t *testing.T) {
		adapter := NewGRPCCLIAdapter(
			"localhost:50051",
			WithTLS(),
		)
		assert.True(t, adapter.config.tls)
		assert.False(t, adapter.config.insecure)
	})

	t.Run("WithInsecure", func(t *testing.T) {
		adapter := NewGRPCCLIAdapter(
			"localhost:50051",
			WithInsecure(),
		)
		assert.True(t, adapter.config.insecure)
	})

	t.Run("WithCert", func(t *testing.T) {
		adapter := NewGRPCCLIAdapter(
			"localhost:50051",
			WithCert("/path/to/cert.pem"),
		)
		assert.True(t, adapter.config.tls)
		assert.Equal(
			t,
			"/path/to/cert.pem",
			adapter.config.certFile,
		)
	})

	t.Run("WithProtoFiles", func(t *testing.T) {
		adapter := NewGRPCCLIAdapter(
			"localhost:50051",
			WithProtoFiles(
				"api.proto", "health.proto",
			),
		)
		assert.Equal(
			t,
			[]string{"api.proto", "health.proto"},
			adapter.config.protoFiles,
		)
	})

	t.Run("WithHeaders", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer token123",
			"X-Request-ID":  "req-abc",
		}
		adapter := NewGRPCCLIAdapter(
			"localhost:50051",
			WithHeaders(headers),
		)
		assert.Equal(
			t,
			"Bearer token123",
			adapter.config.headers["Authorization"],
		)
		assert.Equal(
			t,
			"req-abc",
			adapter.config.headers["X-Request-ID"],
		)
	})

	t.Run("multiple_options_chained", func(t *testing.T) {
		adapter := NewGRPCCLIAdapter(
			"grpc.host:443",
			WithTLS(),
			WithCert("/etc/ssl/cert.pem"),
			WithProtoFiles("service.proto"),
			WithHeaders(map[string]string{
				"Auth": "key",
			}),
		)
		assert.True(t, adapter.config.tls)
		assert.Equal(
			t,
			"/etc/ssl/cert.pem",
			adapter.config.certFile,
		)
		assert.Equal(
			t,
			[]string{"service.proto"},
			adapter.config.protoFiles,
		)
		assert.Equal(
			t,
			"key",
			adapter.config.headers["Auth"],
		)
	})
}

func TestGRPCCLIAdapter_BaseArgs(t *testing.T) {
	tests := []struct {
		name     string
		opts     []GRPCOption
		contains []string
		absent   []string
	}{
		{
			name:     "plaintext_default",
			opts:     nil,
			contains: []string{"-plaintext"},
			absent:   []string{"-insecure", "-cacert"},
		},
		{
			name:   "tls_no_plaintext",
			opts:   []GRPCOption{WithTLS()},
			absent: []string{"-plaintext"},
		},
		{
			name: "insecure_flag",
			opts: []GRPCOption{WithInsecure()},
			contains: []string{
				"-plaintext", "-insecure",
			},
		},
		{
			name: "cert_file",
			opts: []GRPCOption{
				WithCert("/tmp/ca.pem"),
			},
			contains: []string{
				"-cacert", "/tmp/ca.pem",
			},
			absent: []string{"-plaintext"},
		},
		{
			name: "proto_files",
			opts: []GRPCOption{
				WithProtoFiles(
					"a.proto", "b.proto",
				),
			},
			contains: []string{
				"-proto", "a.proto", "b.proto",
			},
		},
		{
			name: "headers",
			opts: []GRPCOption{
				WithHeaders(map[string]string{
					"X-Key": "val",
				}),
			},
			contains: []string{"-H", "X-Key: val"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewGRPCCLIAdapter(
				"localhost:50051", tt.opts...,
			)
			args := adapter.baseArgs()
			for _, want := range tt.contains {
				assert.Contains(t, args, want)
			}
			for _, noWant := range tt.absent {
				assert.NotContains(t, args, noWant)
			}
		})
	}
}

func TestGRPCCLIAdapter_Available_NoServer(
	t *testing.T,
) {
	adapter := NewGRPCCLIAdapter("localhost:19999")
	// grpcurl is likely not installed or server is not
	// running. Available should return false.
	available := adapter.Available(context.Background())
	assert.False(t, available)
}

func TestGRPCCLIAdapter_Close_NoOp(t *testing.T) {
	adapter := NewGRPCCLIAdapter("localhost:50051")
	err := adapter.Close(context.Background())
	assert.NoError(t, err)
}

func TestGRPCCLIAdapter_Close_MultipleCalls(
	t *testing.T,
) {
	adapter := NewGRPCCLIAdapter("localhost:50051")
	for i := 0; i < 5; i++ {
		err := adapter.Close(context.Background())
		assert.NoError(t, err)
	}
}

func TestParseLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple_lines",
			input: "line1\nline2\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "empty_lines_filtered",
			input: "line1\n\nline2\n\n",
			want:  []string{"line1", "line2"},
		},
		{
			name:  "whitespace_trimmed",
			input: "  line1  \n\t line2 \t\n",
			want:  []string{"line1", "line2"},
		},
		{
			name:  "empty_input",
			input: "",
			want:  []string{},
		},
		{
			name:  "only_whitespace",
			input: "  \n\t\n  \n",
			want:  []string{},
		},
		{
			name:  "single_line",
			input: "grpc.health.v1.Health",
			want:  []string{"grpc.health.v1.Health"},
		},
		{
			name: "grpc_service_list",
			input: "grpc.health.v1.Health\n" +
				"grpc.reflection.v1alpha.ServerReflection\n" +
				"myapp.v1.UserService\n",
			want: []string{
				"grpc.health.v1.Health",
				"grpc.reflection.v1alpha.ServerReflection",
				"myapp.v1.UserService",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLines(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseJSONObjects(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single_object",
			input: `{"key":"value"}`,
			want:  []string{`{"key":"value"}`},
		},
		{
			name: "multiple_objects",
			input: `{"a":1}
{"b":2}`,
			want: []string{`{"a":1}`, `{"b":2}`},
		},
		{
			name:  "nested_object",
			input: `{"outer":{"inner":"val"}}`,
			want:  []string{`{"outer":{"inner":"val"}}`},
		},
		{
			name:  "empty_input",
			input: "",
			want:  nil,
		},
		{
			name:  "no_json",
			input: "plain text without json",
			want:  nil,
		},
		{
			name:  "empty_object",
			input: `{}`,
			want:  []string{`{}`},
		},
		{
			name: "objects_with_whitespace",
			input: `  {"a": 1}  
  
  {"b": 2}  `,
			want: []string{`{"a": 1}`, `{"b": 2}`},
		},
		{
			name: "streaming_response",
			input: `{
  "status": "SERVING"
}
{
  "result": "ok"
}`,
			want: []string{
				"{\n  \"status\": \"SERVING\"\n}",
				"{\n  \"result\": \"ok\"\n}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONObjects(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGRPCCLIAdapter_HeadersInitialized(
	t *testing.T,
) {
	// Verify headers map is always initialized even
	// without WithHeaders option.
	adapter := NewGRPCCLIAdapter("localhost:50051")
	assert.NotNil(t, adapter.config.headers)
	assert.Empty(t, adapter.config.headers)
}

func TestGRPCCLIAdapter_ProtoFilesAccumulate(
	t *testing.T,
) {
	adapter := NewGRPCCLIAdapter(
		"localhost:50051",
		WithProtoFiles("a.proto"),
		WithProtoFiles("b.proto", "c.proto"),
	)
	assert.Equal(
		t,
		[]string{"a.proto", "b.proto", "c.proto"},
		adapter.config.protoFiles,
	)
}

func TestGRPCOption_WithHeaders_MergesInto(
	t *testing.T,
) {
	adapter := NewGRPCCLIAdapter(
		"localhost:50051",
		WithHeaders(map[string]string{
			"Key1": "Val1",
		}),
		WithHeaders(map[string]string{
			"Key2": "Val2",
		}),
	)
	assert.Equal(
		t, "Val1", adapter.config.headers["Key1"],
	)
	assert.Equal(
		t, "Val2", adapter.config.headers["Key2"],
	)
}
