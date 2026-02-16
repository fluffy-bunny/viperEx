package viperEx

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDuration mirrors the utils.Duration type: a time.Duration wrapper that
// implements encoding.TextUnmarshaler / encoding.TextMarshaler so that
// mapstructure can decode human-readable strings like "5s" or "2m30s".
type testDuration time.Duration

func (d testDuration) Duration() time.Duration { return time.Duration(d) }
func (d testDuration) String() string          { return time.Duration(d).String() }

func (d testDuration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

func (d *testDuration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", string(text), err)
	}
	*d = testDuration(parsed)
	return nil
}

func (d testDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *testDuration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = testDuration(time.Duration(value))
	case string:
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", value, err)
		}
		*d = testDuration(parsed)
	default:
		return fmt.Errorf("invalid duration: expected string or number, got %T", v)
	}
	return nil
}

// testConfigWithDurations is a config struct using the custom Duration type,
// similar to DistributedEquitableConsumerConfig.
type testConfigWithDurations struct {
	Name                 string       `json:"name"`
	HeartbeatInterval    testDuration `json:"heartbeatInterval"`
	StabilizationWindow  testDuration `json:"stabilizationWindow"`
	RebalanceDebounce    testDuration `json:"rebalanceDebounce"`
	TopologyPollInterval testDuration `json:"topologyPollInterval"`
	SlotClaimTTL         testDuration `json:"slotClaimTtl"`
	Count                int          `json:"count"`
}

// TestUnmarshal_CustomDuration_StringValues verifies that ViperEx.Unmarshal
// correctly decodes human-readable duration strings (e.g. "5s", "30s") into
// a custom Duration type that implements encoding.TextUnmarshaler.
func TestUnmarshal_CustomDuration_StringValues(t *testing.T) {
	settings := map[string]interface{}{
		"name":                 "test-service",
		"heartbeatInterval":    "5s",
		"stabilizationWindow":  "30s",
		"rebalanceDebounce":    "2s",
		"topologyPollInterval": "5s",
		"slotClaimTtl":         "30s",
		"count":                42,
	}

	ve, err := New(settings)
	require.NoError(t, err)

	var cfg testConfigWithDurations
	err = ve.Unmarshal(&cfg)
	require.NoError(t, err, "Unmarshal should handle custom Duration with TextUnmarshaler")

	assert.Equal(t, "test-service", cfg.Name)
	assert.Equal(t, 5*time.Second, cfg.HeartbeatInterval.Duration())
	assert.Equal(t, 30*time.Second, cfg.StabilizationWindow.Duration())
	assert.Equal(t, 2*time.Second, cfg.RebalanceDebounce.Duration())
	assert.Equal(t, 5*time.Second, cfg.TopologyPollInterval.Duration())
	assert.Equal(t, 30*time.Second, cfg.SlotClaimTTL.Duration())
	assert.Equal(t, 42, cfg.Count)
}

// TestUnmarshal_CustomDuration_ComplexStrings verifies parsing of more complex
// duration strings like "2m30s", "1h", "500ms".
func TestUnmarshal_CustomDuration_ComplexStrings(t *testing.T) {
	settings := map[string]interface{}{
		"name":                 "complex-test",
		"heartbeatInterval":    "2m30s",
		"stabilizationWindow":  "1h",
		"rebalanceDebounce":    "500ms",
		"topologyPollInterval": "1m",
		"slotClaimTtl":         "10m30s",
		"count":                1,
	}

	ve, err := New(settings)
	require.NoError(t, err)

	var cfg testConfigWithDurations
	err = ve.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.Equal(t, 2*time.Minute+30*time.Second, cfg.HeartbeatInterval.Duration())
	assert.Equal(t, 1*time.Hour, cfg.StabilizationWindow.Duration())
	assert.Equal(t, 500*time.Millisecond, cfg.RebalanceDebounce.Duration())
	assert.Equal(t, 1*time.Minute, cfg.TopologyPollInterval.Duration())
	assert.Equal(t, 10*time.Minute+30*time.Second, cfg.SlotClaimTTL.Duration())
}

// TestUnmarshal_CustomDuration_NestedStruct verifies that Duration decoding
// works correctly when the Duration fields are inside a nested struct.
func TestUnmarshal_CustomDuration_NestedStruct(t *testing.T) {
	type innerConfig struct {
		Interval testDuration `json:"interval"`
		Timeout  testDuration `json:"timeout"`
	}
	type outerConfig struct {
		Name  string      `json:"name"`
		Inner innerConfig `json:"inner"`
	}

	settings := map[string]interface{}{
		"name": "nested-test",
		"inner": map[string]interface{}{
			"interval": "10s",
			"timeout":  "1m",
		},
	}

	ve, err := New(settings)
	require.NoError(t, err)

	var cfg outerConfig
	err = ve.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.Equal(t, "nested-test", cfg.Name)
	assert.Equal(t, 10*time.Second, cfg.Inner.Interval.Duration())
	assert.Equal(t, 1*time.Minute, cfg.Inner.Timeout.Duration())
}

// TestUnmarshal_CustomDuration_ZeroValue verifies that a zero/missing duration
// field results in a zero Duration value.
func TestUnmarshal_CustomDuration_ZeroValue(t *testing.T) {
	settings := map[string]interface{}{
		"name":  "zero-test",
		"count": 1,
	}

	ve, err := New(settings)
	require.NoError(t, err)

	var cfg testConfigWithDurations
	err = ve.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.Equal(t, time.Duration(0), cfg.HeartbeatInterval.Duration())
	assert.Equal(t, time.Duration(0), cfg.StabilizationWindow.Duration())
}

// TestUnmarshal_StandardTimeDuration_StillWorks verifies that standard
// time.Duration fields (decoded via StringToTimeDurationHookFunc) still
// work alongside the custom Duration type.
func TestUnmarshal_StandardTimeDuration_StillWorks(t *testing.T) {
	type mixedConfig struct {
		CustomTimeout   testDuration  `json:"customTimeout"`
		StandardTimeout time.Duration `json:"standardTimeout"`
	}

	settings := map[string]interface{}{
		"customTimeout":   "15s",
		"standardTimeout": "30s",
	}

	ve, err := New(settings)
	require.NoError(t, err)

	var cfg mixedConfig
	err = ve.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.Equal(t, 15*time.Second, cfg.CustomTimeout.Duration())
	assert.Equal(t, 30*time.Second, cfg.StandardTimeout)
}

// TestUnmarshal_CustomDuration_FromJSON roundtrips a JSON config through
// json.Unmarshal → map → ViperEx.Unmarshal, simulating the real config
// loading pipeline (viper reads JSON → allSettings map → ViperEx.Unmarshal).
func TestUnmarshal_CustomDuration_FromJSON(t *testing.T) {
	jsonConfig := `{
		"name": "json-roundtrip",
		"heartbeatInterval": "5s",
		"stabilizationWindow": "30s",
		"rebalanceDebounce": "2s",
		"topologyPollInterval": "5s",
		"slotClaimTtl": "30s",
		"count": 10
	}`

	var settings map[string]interface{}
	err := json.Unmarshal([]byte(jsonConfig), &settings)
	require.NoError(t, err)

	ve, err := New(settings)
	require.NoError(t, err)

	var cfg testConfigWithDurations
	err = ve.Unmarshal(&cfg)
	require.NoError(t, err)

	assert.Equal(t, "json-roundtrip", cfg.Name)
	assert.Equal(t, 5*time.Second, cfg.HeartbeatInterval.Duration())
	assert.Equal(t, 30*time.Second, cfg.StabilizationWindow.Duration())
	assert.Equal(t, 2*time.Second, cfg.RebalanceDebounce.Duration())
	assert.Equal(t, 5*time.Second, cfg.TopologyPollInterval.Duration())
	assert.Equal(t, 30*time.Second, cfg.SlotClaimTTL.Duration())
	assert.Equal(t, 10, cfg.Count)
}
