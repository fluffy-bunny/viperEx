// Copyright © 2020 Herb Stahl <ghstahl@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// ViperEx adds some missing gap items from the awesome Viper project is a application configuration system.

package viperEx

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const defaultKeyDelimiter = "."

func newChangeAllKeysToLowerCase(m map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			newMap[strings.ToLower(k)] = newChangeAllKeysToLowerCase(val)
		case []interface{}:
			var newSlice []interface{}
			for _, item := range val {
				if reflect.TypeOf(item).Kind() == reflect.Map {
					newSlice = append(newSlice, newChangeAllKeysToLowerCase(item.(map[string]interface{})))
				} else {
					newSlice = append(newSlice, item)
				}
			}
			newMap[strings.ToLower(k)] = newSlice
		default:
			newMap[strings.ToLower(k)] = val
		}
	}
	return newMap
}

// WithEnvPrefix sets the prefix for environment variables
func WithEnvPrefix(envPrefix string) func(*ViperEx) error {
	return func(v *ViperEx) error {
		v.EnvPrefix = envPrefix + "_"
		return nil
	}
}

// WithDelimiter sets the delimiter for keys
func WithDelimiter(delimiter string) func(*ViperEx) error {
	return func(v *ViperEx) error {
		v.KeyDelimiter = delimiter
		return nil
	}
}

// New creates a new ViperEx instance with optional options
func New(allsettings map[string]interface{}, options ...func(*ViperEx) error) (*ViperEx, error) {
	changeAllKeysToLowerCase(allsettings)
	changeStringArrayToInterfaceArray(allsettings)
	changeStringMapStringToStringMapInterface(allsettings)
	viperEx := &ViperEx{
		KeyDelimiter: defaultKeyDelimiter,
		AllSettings:  newChangeAllKeysToLowerCase(allsettings),
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

// ViperEx type
type ViperEx struct {
	KeyDelimiter string
	AllSettings  map[string]interface{}
	EnvPrefix    string
}

// UpdateFromEnv will find potential ENV candidates to merge in
func (ve *ViperEx) UpdateFromEnv() error {
	potential := ve.getPotentialEnvVariables()
	for key, value := range potential {
		ve.UpdateDeepPath(key, value)
	}
	return nil
}

// Find will return the interface to the data if it exists
func (ve *ViperEx) Find(key string) interface{} {
	lcaseKey := strings.ToLower(key)
	path := strings.Split(lcaseKey, ve.KeyDelimiter)

	lastKey := strings.ToLower(path[len(path)-1])

	fmt.Println(lastKey)
	path = path[0 : len(path)-1]
	if len(lastKey) == 0 {
		return nil
	}

	deepestEntity := ve.deepSearch(ve.AllSettings, path)
	deepestMap, ok := deepestEntity.(map[string]interface{})
	if ok {
		return deepestMap[lastKey]
	}

	deepestArray, ok := deepestEntity.([]interface{})
	if ok {
		// lastKey has to be a num
		idx, err := strconv.Atoi(lastKey)
		if err == nil {
			return deepestArray[idx]
		}
	}

	return nil
}

// UpdateDeepPath will update the value if it exists
func (ve *ViperEx) UpdateDeepPath(key string, value interface{}) {
	lcaseKey := strings.ToLower(key)
	path := strings.Split(lcaseKey, ve.KeyDelimiter)

	lastKey := strings.ToLower(path[len(path)-1])

	path = path[0 : len(path)-1]
	if len(lastKey) == 0 {
		return
	}

	deepestEntity := ve.deepSearch(ve.AllSettings, path)
	deepestMap, ok := deepestEntity.(map[string]interface{})
	if ok {
		// set innermost value
		_, ok := deepestMap[lastKey]
		if ok {
			deepestMap[lastKey] = value
		}
	} else {
		// is this an array
		deepestArray, ok := deepestEntity.([]interface{})
		if ok {
			// lastKey has to be a num
			idx, err := strconv.Atoi(lastKey)
			if err == nil {
				if idx < len(deepestArray) && idx >= 0 {
					deepestArray[idx] = value
				}
			}
		}
	}
}
func (ve *ViperEx) getPotentialEnvVariables() map[string]string {
	var result map[string]string
	result = make(map[string]string)
	for _, element := range os.Environ() {
		var index = strings.Index(element, "=")
		key := element[0:index]
		// check for prefix
		if len(ve.EnvPrefix) > 0 {
			if strings.HasPrefix(key, ve.EnvPrefix) {
				key = key[len(ve.EnvPrefix):]
			}
		}
		value := element[index+1:]
		if strings.Contains(key, ve.KeyDelimiter) {
			result[key] = value
		}
	}
	return result
}

func (ve *ViperEx) deepSearch(m map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return m
	}
	var currentPath string
	var stepArray = false
	var currentArray []interface{}
	var currentEntity interface{}
	for _, k := range path {
		if len(currentPath) == 0 {
			currentPath = k
		} else {
			currentPath = fmt.Sprintf("%v.%v", currentPath, k)
		}
		if stepArray {
			idx, err := strconv.Atoi(k)
			if err != nil {
				log.Error().Err(err).Msgf("No such path exists, must be an array idx: %v", currentPath)
				return nil
			}
			if len(currentArray) <= idx {
				log.Error().Msgf("No such path exists: %v", currentPath)
				return nil
			}
			m3, ok := currentArray[idx].(map[string]interface{})
			if !ok {
				log.Error().Msgf("No such path exists: %v, error in mapping to a map[string]interface{}", currentPath)
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

// defaultDecoderConfig returns default mapsstructure.DecoderConfig with suppot
// of time.Duration values & string slices
func defaultDecoderConfig(output interface{}, opts ...viper.DecoderConfigOption) *mapstructure.DecoderConfig {
	c := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
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
func changeStringMapStringToStringMapInterface(m map[string]interface{}) {
	var currentKeys []string
	for key := range m {
		currentKeys = append(currentKeys, key)
	}
	for _, key := range currentKeys {
		vv, ok := m[key].(map[string]string)
		if ok {
			m2 := make(map[string]interface{})
			for k, v := range vv {
				m2[k] = v
			}
			m[key] = m2
		} else {
			v2, ok := m[key].(map[string]interface{})
			if ok {
				changeStringMapStringToStringMapInterface(v2)
			}
		}
	}
}
func changeStringArrayToInterfaceArray(m map[string]interface{}) {
	var currentKeys []string
	for key := range m {
		currentKeys = append(currentKeys, key)
	}

	for _, key := range currentKeys {
		vv, ok := m[key].([]string)
		if ok {
			m2 := make([]interface{}, 0)
			for idx := range vv {
				v := vv[idx]
				m2 = append(m2, &v)
			}
			m[key] = m2
		} else {
			v2, ok := m[key].(map[string]interface{})
			if ok {
				changeStringArrayToInterfaceArray(v2)
			}
		}
	}
}

func changeAllKeysToLowerCase(m map[string]interface{}) {
	var lcMap = make(map[string]interface{})
	var currentKeys []string
	for key, value := range m {
		currentKeys = append(currentKeys, key)
		lcMap[strings.ToLower(key)] = value
	}
	// delete original values
	for _, k := range currentKeys {
		delete(m, k)
	}
	// put the lowercase ones in the original map
	for key, value := range lcMap {
		m[key] = value
		vMap, ok := value.(map[string]interface{})
		if ok {
			// if the current value is a map[string]interface{}, keep going
			changeAllKeysToLowerCase(vMap)
		}
	}
}
