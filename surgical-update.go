// Copyright © 2020 Herb Stahl <ghstahl@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// ViperEx adds some missing gap items from the awesome Viper project is a application configuration system.

package viperEx

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

const defaultKeyDelimiter = "__"

func New(keyDelimiter string) *ViperEx {
	if len(keyDelimiter) == 0 {
		return &ViperEx{
			KeyDelimiter: defaultKeyDelimiter,
		}
	}
	return &ViperEx{
		KeyDelimiter: keyDelimiter,
	}
}

type ViperEx struct {
	KeyDelimiter string
}

// UpdateFromEnv will find potential ENV candidates to merge in
func (ve *ViperEx) UpdateFromEnv(dst map[string]interface{}) error {
	potential := ve.getPotentialEnvVariables()
	for key, value := range potential {
		ve.SurgicalUpdate(key, value, dst)
	}
	return nil
}

// Find will return the interface to the data if it exists
func (ve *ViperEx) Find(key string, src map[string]interface{}) interface{} {
	lcaseKey := strings.ToLower(key)
	path := strings.Split(lcaseKey, ve.KeyDelimiter)

	lastKey := strings.ToLower(path[len(path)-1])

	fmt.Println(lastKey)
	path = path[0 : len(path)-1]
	if len(lastKey) == 0 {
		return nil
	}

	deepestEntity := ve.deepSearch(src, path)
	deepestMap, ok := deepestEntity.(map[string]interface{})
	if ok {
		return deepestMap[lastKey]
	} else {
		deepestArray, ok := deepestEntity.([]interface{})
		if ok {
			// lastKey has to be a num
			idx, err := strconv.Atoi(lastKey)
			if err == nil {
				return deepestArray[idx]
			}
		}
	}
	return nil

}

// SurgicalUpdate will update the value if it exists
func (ve *ViperEx) SurgicalUpdate(key string, value interface{}, dst map[string]interface{}) {
	lcaseKey := strings.ToLower(key)
	path := strings.Split(lcaseKey, ve.KeyDelimiter)

	lastKey := strings.ToLower(path[len(path)-1])

	path = path[0 : len(path)-1]
	if len(lastKey) == 0 {
		return
	}

	deepestEntity := ve.deepSearch(dst, path)
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
				deepestArray[idx] = value
			}
		}
	}
}
func (ve *ViperEx) getPotentialEnvVariables() map[string]string {
	var result map[string]string
	result = make(map[string]string)
	for _, element := range os.Environ() {
		variable := strings.Split(element, "=")
		if strings.Contains(variable[0], ve.KeyDelimiter) {
			result[variable[0]] = variable[1]
		}
	}
	return result
}

func (ve *ViperEx) deepSearch(m map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return m
	}
	var currentPath string
	var stepArray bool = false
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
