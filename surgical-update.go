// Copyright © 2020 Herb Stahl <ghstahl@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// ViperEx adds some missing gap items from the awesome Viper project is a application configuration system.

package viperEx

import (
	"os"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const defaultKeyDelimiter = "."

// normalizeValue applies type normalization to a single value:
// lowercases map keys, converts []string→[]interface{}, and
// converts map[string]string→map[string]interface{}.
func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return normalizeSettings(val)
	case map[string]string:
		newMap := make(map[string]interface{}, len(val))
		for k, v := range val {
			newMap[strings.ToLower(k)] = v
		}
		return newMap
	case []interface{}:
		newSlice := make([]interface{}, len(val))
		for i, item := range val {
			newSlice[i] = normalizeValue(item)
		}
		return newSlice
	case []string:
		newSlice := make([]interface{}, len(val))
		for i, item := range val {
			newSlice[i] = item
		}
		return newSlice
	default:
		return v
	}
}

// normalizeSettings returns a deep copy of m with all map keys lowercased,
// []string converted to []interface{}, and map[string]string converted to
// map[string]interface{}. The original map is not modified.
func normalizeSettings(m map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{}, len(m))
	for k, v := range m {
		newMap[strings.ToLower(k)] = normalizeValue(v)
	}
	return newMap
}

// WithEnvPrefix sets the prefix for environment variables.
// The prefix is automatically separated from keys by an underscore.
// For example, WithEnvPrefix("APP") matches env vars like APP_some__key.
func WithEnvPrefix(envPrefix string) func(*ViperEx) error {
	return func(v *ViperEx) error {
		envPrefix = strings.TrimRight(envPrefix, "_")
		if len(envPrefix) == 0 {
			return nil
		}
		v.EnvPrefix = envPrefix + "_"
		return nil
	}
}

// WithDelimiter sets the key path delimiter used to separate path segments.
// The default delimiter is ".".
func WithDelimiter(delimiter string) func(*ViperEx) error {
	return func(v *ViperEx) error {
		v.KeyDelimiter = delimiter
		return nil
	}
}

// New creates a new ViperEx instance with optional options.
// The provided allsettings map is not modified; a normalized deep copy is used internally.
func New(allsettings map[string]interface{}, options ...func(*ViperEx) error) (*ViperEx, error) {
	viperEx := &ViperEx{
		KeyDelimiter: defaultKeyDelimiter,
		AllSettings:  normalizeSettings(allsettings),
	}
	var err error
	for _, option := range options {
		err = option(viperEx)
		if err != nil {
			return nil, err
		}
	}
	return viperEx, nil
}

// ViperEx extends spf13/viper with surgical deep-path updates for nested
// maps, arrays, and mixed structures using a configurable key delimiter.
type ViperEx struct {
	// KeyDelimiter separates path segments in deep-path keys (default ".").
	KeyDelimiter string
	// AllSettings holds the normalized configuration map.
	AllSettings map[string]interface{}
	// EnvPrefix, when set, filters environment variables to only those
	// starting with this prefix. Set via WithEnvPrefix.
	EnvPrefix string
}

// UpdateFromEnv finds environment variables whose keys contain the
// configured delimiter and merges their values into the settings.
// If an EnvPrefix is configured, only matching env vars are considered.
func (ve *ViperEx) UpdateFromEnv() {
	potential := ve.getPotentialEnvVariables()
	for key, value := range potential {
		ve.UpdateDeepPath(key, value)
	}
}

// Find returns the value at the given deep-path key and true if found,
// or nil and false if the path does not exist.
func (ve *ViperEx) Find(key string) (interface{}, bool) {
	lcaseKey := strings.ToLower(key)
	path := strings.Split(lcaseKey, ve.KeyDelimiter)

	lastKey := strings.ToLower(path[len(path)-1])

	path = path[0 : len(path)-1]
	if len(lastKey) == 0 {
		return nil, false
	}

	deepestEntity := ve.deepSearch(ve.AllSettings, path)
	deepestMap, ok := deepestEntity.(map[string]interface{})
	if ok {
		val, exists := deepestMap[lastKey]
		return val, exists
	}

	deepestArray, ok := deepestEntity.([]interface{})
	if ok {
		// lastKey has to be a num
		idx, err := strconv.Atoi(lastKey)
		if err == nil && idx >= 0 && idx < len(deepestArray) {
			return deepestArray[idx], true
		}
	}

	return nil, false
}

// UpdateDeepPath updates the value at the given deep-path key, returning
// true if the path was found and updated, or false if the path does not exist.
func (ve *ViperEx) UpdateDeepPath(key string, value interface{}) bool {
	lcaseKey := strings.ToLower(key)
	path := strings.Split(lcaseKey, ve.KeyDelimiter)

	lastKey := strings.ToLower(path[len(path)-1])

	path = path[0 : len(path)-1]
	if len(lastKey) == 0 {
		return false
	}

	deepestEntity := ve.deepSearch(ve.AllSettings, path)
	deepestMap, ok := deepestEntity.(map[string]interface{})
	if ok {
		// set innermost value
		_, ok := deepestMap[lastKey]
		if ok {
			deepestMap[lastKey] = value
			return true
		}
		return false
	}
	// is this an array
	deepestArray, ok := deepestEntity.([]interface{})
	if ok {
		// lastKey has to be a num
		idx, err := strconv.Atoi(lastKey)
		if err == nil {
			if idx < len(deepestArray) && idx >= 0 {
				deepestArray[idx] = value
				return true
			}
		}
	}
	return false
}
func (ve *ViperEx) getPotentialEnvVariables() map[string]string {
	var result map[string]string
	result = make(map[string]string)
	for _, element := range os.Environ() {
		var index = strings.Index(element, "=")
		key := element[0:index]
		// check for prefix
		if len(ve.EnvPrefix) > 0 {
			if !strings.HasPrefix(key, ve.EnvPrefix) {
				continue
			}
			key = key[len(ve.EnvPrefix):]
		}
		value := element[index+1:]
		if strings.Contains(key, ve.KeyDelimiter) {
			result[key] = value
		}
	}
	return result
}

// deepSearch walks the settings tree along the given path segments.
// It supports maps and arrays but does not support nested arrays-of-arrays;
// when an array element is itself an array, the search returns nil.
func (ve *ViperEx) deepSearch(m map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return m
	}
	var stepArray = false
	var currentArray []interface{}
	var currentEntity interface{}
	for _, k := range path {
		if stepArray {
			idx, err := strconv.Atoi(k)
			if err != nil {
				return nil
			}
			if len(currentArray) <= idx {
				return nil
			}
			m3, ok := currentArray[idx].(map[string]interface{})
			if !ok {
				return nil
			}
			// continue search from here
			m = m3
			currentEntity = m
			stepArray = false // don't support arrays of arrays
		} else {
			m2, ok := m[k]
			if !ok {
				// intermediate key does not exist
				return nil
			}
			m3, ok := m2.(map[string]interface{})
			if !ok {
				// is this an array
				m4, ok := m2.([]interface{})
				if ok {
					// continue search from here
					currentArray = m4
					currentEntity = currentArray
					stepArray = true
					m3 = nil
				} else {
					// intermediate key is a value
					return nil
				}
			} else {
				// continue search from here
				m = m3
				currentEntity = m
			}
		}
	}

	return currentEntity
}

// code copied from the viper project

// defaultDecoderConfig returns default mapstructure.DecoderConfig with support
// of time.Duration values & string slices
func defaultDecoderConfig(output interface{}, opts ...viper.DecoderConfigOption) *mapstructure.DecoderConfig {
	c := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.TextUnmarshallerHookFunc(),
		),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Unmarshal to struct
func (ve *ViperEx) Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error {
	return decode(ve.AllSettings, defaultDecoderConfig(rawVal, opts...))
}

// A wrapper around mapstructure.Decode that mimics the WeakDecode functionality
func decode(input interface{}, config *mapstructure.DecoderConfig) error {
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

