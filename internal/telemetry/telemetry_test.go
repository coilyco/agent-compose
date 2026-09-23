package telemetry

import (
	"slices"
	"testing"
)

var endpoint = &Config{OTLPMetricsEndpoint: "http://collector.example:4318/v1/metrics"}

func TestEnabledExportsMetricsOnlyWithSeatAndShadow(t *testing.T) {
	plan := Environment(endpoint, "claude", "platform-eng", "yx46", "")
	want := map[string]string{
		"CLAUDE_CODE_ENABLE_TELEMETRY":        "1",
		"OTEL_METRICS_EXPORTER":               "otlp",
		"OTEL_LOGS_EXPORTER":                  "none",
		"OTEL_TRACES_EXPORTER":                "none",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": endpoint.OTLPMetricsEndpoint,
		"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL": DefaultProtocol,
		"OTEL_RESOURCE_ATTRIBUTES":            "seat=platform-eng,shadow=yx46",
	}
	for name, value := range want {
		if plan.Set[name] != value {
			t.Errorf("%s = %q, want %q", name, plan.Set[name], value)
		}
	}
	// A parent seat that logged prompts must not hand that to its child.
	for _, name := range []string{"OTEL_LOG_USER_PROMPTS", "OTEL_LOG_TOOL_DETAILS", "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT"} {
		if !slices.Contains(plan.Unset, name) {
			t.Errorf("%s is not cleared", name)
		}
	}
}

func TestKillSwitchAndMissingEndpointClearEverything(t *testing.T) {
	for name, plan := range map[string]Plan{
		"kill switch": Environment(endpoint, "claude", "platform-eng", "yx46", "OFF"),
		"no endpoint": Environment(&Config{}, "claude", "platform-eng", "yx46", ""),
		"no config":   Environment(nil, "claude", "platform-eng", "", ""),
	} {
		if len(plan.Set) != 0 {
			t.Errorf("%s sets %v", name, plan.Set)
		}
		if !slices.Equal(plan.Unset, managed) {
			t.Errorf("%s clears %v, want every managed variable", name, plan.Unset)
		}
	}
}

func TestOtherHarnessesAreUntouched(t *testing.T) {
	if plan := Environment(endpoint, "codex", "platform-eng", "yx46", ""); len(plan.Set)+len(plan.Unset) != 0 {
		t.Fatalf("codex plan = %+v", plan)
	}
}

func TestAttributeValuesAreEncodedAndShadowIsOptional(t *testing.T) {
	plan := Environment(endpoint, "claude", "a b,c=d", "", "")
	if got := plan.Set["OTEL_RESOURCE_ATTRIBUTES"]; got != "seat=a%20b%2Cc%3Dd" {
		t.Fatalf("attributes = %q", got)
	}
}

func TestValidate(t *testing.T) {
	for _, bad := range []Config{
		{OTLPMetricsEndpoint: "collector.example:4318"},
		{OTLPMetricsEndpoint: "http://collector.example:4318/v1/metrics", Protocol: "udp"},
	} {
		if err := bad.Validate(); err == nil {
			t.Errorf("%+v validated", bad)
		}
	}
	good := Config{OTLPMetricsEndpoint: "http://collector.example:4318/v1/metrics", Protocol: "http/protobuf"}
	if err := good.Validate(); err != nil {
		t.Errorf("valid config rejected: %v", err)
	}
}
