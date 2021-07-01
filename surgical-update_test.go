// Copyright © 2020 Herb Stahl <ghstahl@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// ViperEx adds some missing gap items from the awesome Viper project is a application configuration system.

package viperEx

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type NestedMap struct {
	Eggs map[string]Egg
}
type SettingsWithNestedMap struct {
	Name      string
	NestedMap *NestedMap
}
type Nest struct {
	Name       string
	CountInt   int
	CountInt16 int16
	MasterEgg  Egg
	Eggs       []Egg
}
type ValueContainer struct {
	Value interface{}
}

func (vc *ValueContainer) GetString() (string, bool) {
	value, ok := vc.Value.(string)
	return value, ok
}

type Egg struct {
	Weight      int32
	SomeValues  []ValueContainer
	SomeStrings []string
}
type Settings struct {
	Name string
	Nest *Nest
}

func init() {
	chdirToTestFolder()
}

const keyDelim = "__"

func ReadAppsettings(rootPath string) (*viper.Viper, error) {
	var err error
	environment := os.Getenv("APPLICATION_ENVIRONMENT")
	myViper := viper.NewWithOptions(viper.KeyDelimiter(keyDelim))
	// Environment Variables override everything.
	myViper.AutomaticEnv()
	myViper.SetConfigType("json")
	configFile := "appsettings.json"
	configPath := path.Join(rootPath, configFile)
	viper.SetConfigFile(configPath)
	err = myViper.ReadInConfig()
	if err == nil {
		return nil, err
	}
	configFile = "appsettings." + environment + ".json"
	myViper.SetConfigFile(configPath)
	err = myViper.MergeInConfig()
	return myViper, err
}

func getConfigPath() string {
	var configPath string
	_, err := os.Stat("./settings")
	if !os.IsNotExist(err) {
		configPath, _ = filepath.Abs("./settings")
		log.Info().Str("path", configPath).Msg("Configuration Root Folder")
	}
	return configPath
}
func chdirToTestFolder() {
	_, filename, _, _ := runtime.Caller(0)
	// The ".." may change depending on you folder structure
	dir := path.Join(path.Dir(filename), ".")
	fmt.Println(filename)
	fmt.Println(dir)
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func TestViperExEnvUpdate(t *testing.T) {
	configPath := getConfigPath()
	myViper, err := ReadAppsettings(configPath)
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}
	allSettings := myViper.AllSettings()
	fmt.Println(PrettyJSON(allSettings))

	myViperEx, err := New(allSettings, func(ve *ViperEx) error {
		ve.KeyDelimiter = keyDelim
		return nil
	})
	envs := map[string]string{

		"APPLICATION_ENVIRONMENT":             "Test",
		"nest__Eggs__1__Weight":               "5555",
		"nest__Eggs__1__SomeValues__1__Value": "Heidi",
		"nest__Eggs__1__SomeStrings__1":       "Zep",
	}
	for k, v := range envs {
		os.Setenv(k, v)
	}
	os.Setenv("APPLICATION_ENVIRONMENT", "Test")

	err = myViperEx.UpdateFromEnv()
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}
	for k := range envs {
		os.Remove(k)
	}

	settings := Settings{}
	err = myViperEx.Unmarshal(&settings)
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "Heidi", settings.Nest.Eggs[1].SomeValues[1].Value)
	assert.Equal(t, "Zep", settings.Nest.Eggs[1].SomeStrings[1])
	assert.Equal(t, int32(5555), settings.Nest.Eggs[1].Weight)
}
func TestViperSurgicalUpdate_NestedMap(t *testing.T) {
	configPath := getConfigPath()
	myViper, err := ReadAppsettings(configPath)
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}
	allSettings := myViper.AllSettings()
	fmt.Println(PrettyJSON(allSettings))

	myViperEx, err := New(allSettings, func(ve *ViperEx) error {
		ve.KeyDelimiter = "__"
		return nil
	})

	envs := map[string]interface{}{
		"name":                                         "bowie",
		"nestedMap__Eggs__bob__Weight":                 1234,
		"nestedMap__Eggs__bob__Weight__":               1234,
		"nestedMap__Eggs__bob__SomeValues__1__Value":   "abcd",
		"nestedMap__Eggs__bob__SomeStrings__1":         "abcd",
		"nestedMap__Eggs__bob__SomeStrings__1__":       "abcd",
		"nestedMap__Eggs__junk__SomeStrings__1__":      "abcd",
		"nestedMap__Eggs__junk__SomeStrings__1__Value": "abcd",
	}
	for k, v := range envs {
		myViperEx.UpdateDeepPath(k, v)
	}

	fmt.Println(PrettyJSON(allSettings))

	settings := SettingsWithNestedMap{}
	err = myViperEx.Unmarshal(&settings)
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "bowie", settings.Name)
	assert.Equal(t, int32(1234), settings.NestedMap.Eggs["bob"].Weight)
	assert.Equal(t, "abcd", settings.NestedMap.Eggs["bob"].SomeValues[1].Value)
	assert.Equal(t, "abcd", settings.NestedMap.Eggs["bob"].SomeStrings[1])
	_, ok := allSettings["junk"]
	assert.False(t, ok)

	name := myViperEx.Find("name")
	assert.NotNil(t, name)

	nestJunk := myViperEx.Find("nestedMap__Eggs__junk")
	assert.Nil(t, nestJunk)

	nestJunk = myViperEx.Find("nestedMap__Eggs__junk__SomeStrings__1__Value")
	assert.Nil(t, nestJunk)
}
func TestViperSurgicalUpdate(t *testing.T) {
	configPath := getConfigPath()
	myViper, err := ReadAppsettings(configPath)
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}
	allSettings := myViper.AllSettings()
	fmt.Println(PrettyJSON(allSettings))

	myViperEx, err := New(allSettings, func(ve *ViperEx) error {
		ve.KeyDelimiter = "__"
		return nil
	})
	envs := map[string]interface{}{
		"nest__MasterEgg__Weight":             1,
		"nest__Eggs__0__Weight":               1234,
		"nest__Eggs__0__Weight__":             1234,
		"nest__Eggs__0__SomeValues__1__Value": "abcd",
		"nest__Eggs__0__SomeStrings__1":       "abcd",
		"nest__Eggs__0__SomeStrings__1__":     "abcd",
		"junk__A":                             "abcd",
		"nest__junk":                          "abcd",
	}
	for k, v := range envs {
		myViperEx.UpdateDeepPath(k, v)
	}

	fmt.Println(PrettyJSON(allSettings))

	settings := Settings{}
	err = myViperEx.Unmarshal(&settings)
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "bob", settings.Name)
	assert.Equal(t, int32(1), settings.Nest.MasterEgg.Weight)
	assert.Equal(t, int32(1234), settings.Nest.Eggs[0].Weight)
	assert.Equal(t, "abcd", settings.Nest.Eggs[0].SomeValues[1].Value)
	assert.Equal(t, "abcd", settings.Nest.Eggs[0].SomeStrings[1])
	_, ok := allSettings["junk"]
	assert.False(t, ok)

	name := myViperEx.Find("name")
	assert.NotNil(t, name)

	nestJunk := myViperEx.Find("nest__junk")
	assert.Nil(t, nestJunk)

	var item interface{}

	item = myViperEx.Find("nest__Eggs")
	assert.NotNil(t, item)

	item = myViperEx.Find("nest__Eggs")
	assert.NotNil(t, item)

	item = myViperEx.Find("nest__Eggs__")
	assert.Nil(t, item)

	item = myViperEx.Find("nest__Eggs__0__SomeStrings__1")
	assert.NotNil(t, item)

	item = myViperEx.Find("nest__Eggs__0__SomeStrings__1__")
	assert.Nil(t, item)

	item = myViperEx.Find("nest__Eggs__0__Junk__1")
	assert.Nil(t, item)

	item = myViperEx.Find("nest__junk__0__Junk__1")
	assert.Nil(t, item)

	item = myViperEx.Find("nest__junk__0")
	assert.Nil(t, item)
}

// PrettyJSON to string
func PrettyJSON(obj interface{}) string {
	jsonBytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}

// JSON from object
func JSON(obj interface{}) string {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}
