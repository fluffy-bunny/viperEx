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

func (ve *ViperEx) UpdateFromEnv(dst map[string]interface{}) error {
	potential := ve.getPotentialEnvVariables()
	for key, value := range potential {
		ve.SurgicalUpdate(key, value, dst)
	}
	return nil
}

func (ve *ViperEx) SurgicalUpdate(key string, value interface{}, dst map[string]interface{}) {

	lcaseKey := strings.ToLower(key)
	path := strings.Split(lcaseKey, ve.KeyDelimiter)

	lastKey := strings.ToLower(path[len(path)-1])

	fmt.Println(lastKey)
	path = path[0 : len(path)-1]
	if len(lastKey) == 0 {
		// we are targeting an array that contains a primitive
		deepestArray, idx := ve.deepSearchArray(dst, path)
		if deepestArray != nil && idx > -1 {
			deepestArray[idx] = value
		}
	} else {
		deepestMap := ve.deepSearch(dst, path)
		if deepestMap == nil {
			fmt.Println("nothign")
		} else {
			// set innermost value
			deepestMap[lastKey] = value
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
func (ve *ViperEx) deepSearchArray(m map[string]interface{}, path []string) ([]interface{}, int) {
	var currentPath string
	var stepArray bool = false
	var currentArray []interface{}
	var currentIdx int = -1
	var err error
	pathDepth := len(path)
	for currentPathIdx, k := range path {
		if len(currentPath) == 0 {
			currentPath = k
		} else {
			currentPath = fmt.Sprintf("%v.%v", currentPath, k)
		}
		if stepArray {
			currentIdx, err = strconv.Atoi(k)
			if err != nil {
				log.Error().Err(err).Msgf("No such path exists, must be an array idx: %v", currentPath)
				return nil, -1
			}
			if len(currentArray) <= currentIdx {
				log.Error().Msgf("No such path exists: %v", currentPath)
				return nil, -1
			}
			m2 := currentArray[currentIdx]
			m3, ok := m2.(map[string]interface{})
			if !ok {
				// is this an array
				m4, ok := m2.([]interface{})
				if ok {
					currentArray = m4
					stepArray = true
					m3 = nil
				} else {
					if currentPathIdx == pathDepth-1 {
						// end of the line
						continue
					} else {
						// we have a problem
						log.Error().Msgf("No such path exists: %v", currentPath)
						return nil, -1
					}

				}
			}
			// continue search from here
			m = m3
			stepArray = false // don't support arrays of arrays
		} else {
			m2, ok := m[k]
			if !ok {
				// intermediate key does not exist
				// => create it and continue from there
				m3 := make(map[string]interface{})
				m[k] = m3
				m = m3
				continue
			}
			m3, ok := m2.(map[string]interface{})
			if !ok {
				// is this an array
				m4, ok := m2.([]interface{})
				if ok {
					currentArray = m4
					stepArray = true
					m3 = nil
				} else {
					// intermediate key is a value
					// => replace with a new map
					m3 = make(map[string]interface{})
					m[k] = m3

				}
			}
			// continue search from here
			m = m3

		}
	}
	return currentArray, currentIdx
}

func (ve *ViperEx) deepSearch(m map[string]interface{}, path []string) map[string]interface{} {
	var currentPath string
	var stepArray bool = false
	var currentArray []interface{}
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
			stepArray = false // don't support arrays of arrays
		} else {
			m2, ok := m[k]
			if !ok {
				// intermediate key does not exist
				// => create it and continue from there
				m3 := make(map[string]interface{})
				m[k] = m3
				m = m3
				continue
			}
			m3, ok := m2.(map[string]interface{})
			if !ok {
				// is this an array
				m4, ok := m2.([]interface{})
				if ok {
					currentArray = m4
					stepArray = true
					m3 = nil
				} else {
					// intermediate key is a value
					// => replace with a new map
					m3 = make(map[string]interface{})
					m[k] = m3

				}
			}
			// continue search from here
			m = m3

		}
	}
	return m
}
