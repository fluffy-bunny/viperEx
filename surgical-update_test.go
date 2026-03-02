// Copyright © 2020 Herb Stahl <ghstahl@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// ViperEx adds some missing gap items from the awesome Viper project is a application configuration system.

package viperEx

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type NestedMap struct {
	Eggs map[string]Egg
}
type SettingsWithNestedMap struct {
	Name      string
	NestedMap *NestedMap
	MasterEgg Egg
}
type Nest struct {
	Name       string
	CountInt   int
	CountInt16 int16
	MasterEgg  Egg
	Eggs       []Egg
	Tags       []string
}
type ValueContainer struct {
	Value interface{} `json:"value"`
}

func (vc *ValueContainer) GetString() (string, bool) {
	value, ok := vc.Value.(string)
	return value, ok
}

type Egg struct {
	Weight      int32            `json:"weight"`
	SomeValues  []ValueContainer `json:"somevalues"`
	SomeStrings []string         `json:"somestrings"`
	Name        string           `json:"name"`
}
type Settings struct {
	Name        string
	Nest        *Nest
	SomeStrings []string
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
	myViper.SetConfigFile(configPath)
	err = myViper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	if len(environment) > 0 {
		configFile = "appsettings." + environment + ".json"
		configPath = path.Join(rootPath, configFile)
		myViper.SetConfigFile(configPath)
		_ = myViper.MergeInConfig() // optional: env-specific config may not exist
	}
	return myViper, nil
}

func getConfigPath() string {
	var configPath string
	_, err := os.Stat("./settings")
	if !os.IsNotExist(err) {
		configPath, _ = filepath.Abs("./settings")
	}
	return configPath
}
func chdirToTestFolder() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), ".")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func TestViperExEnvUpdate(t *testing.T) {
	configPath := getConfigPath()
	myViper, err := ReadAppsettings(configPath)
	require.NoError(t, err)

	allSettings := myViper.AllSettings()
	t.Log(prettyJSON(allSettings))

	myViperEx, err := New(allSettings, func(ve *ViperEx) error {
		ve.KeyDelimiter = keyDelim
		return nil
	})
	require.NoError(t, err)

	expectedURI := "https://www.blah.com/?ssl=true&a=b&c=d"
	envs := map[string]string{
		"APPLICATION_ENVIRONMENT":             "Test",
		"nest__Eggs__1__Weight":               "5555",
		"nest__Eggs__1__SomeValues__1__Value": "Heidi",
		"nest__Eggs__1__SomeStrings__1":       "Zep",
		"nest__masteregg__name":               expectedURI,
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}

	myViperEx.UpdateFromEnv()
	t.Log(prettyJSON(allSettings))

	settings := Settings{}
	err = myViperEx.Unmarshal(&settings)
	require.NoError(t, err)

	assert.Equal(t, expectedURI, settings.Nest.MasterEgg.Name)
	assert.Equal(t, "straw", settings.Nest.Name)
	assert.Equal(t, "Heidi", settings.Nest.Eggs[1].SomeValues[1].Value)
	assert.Equal(t, "Zep", settings.Nest.Eggs[1].SomeStrings[1])
	assert.Equal(t, int32(5555), settings.Nest.Eggs[1].Weight)
}
func TestViperSurgicalUpdate_URIWithArgs(t *testing.T) {
	configPath := getConfigPath()
	myViper, err := ReadAppsettings(configPath)
	require.NoError(t, err)

	allSettings := myViper.AllSettings()
	t.Log(prettyJSON(allSettings))

	myViperEx, err := New(allSettings, func(ve *ViperEx) error {
		ve.KeyDelimiter = "__"
		return nil
	})
	require.NoError(t, err)

	expectedURI := "https://www.blah.com/?ssl=true&a=b&c=d"
	envs := map[string]interface{}{
		"name": expectedURI,
		"nestedMap__Eggs__bob__SomeValues__1__Value": expectedURI,
		"MasterEgg__name": expectedURI,
	}
	for k, v := range envs {
		myViperEx.UpdateDeepPath(k, v)
	}
	t.Log(prettyJSON(allSettings))

	settings := SettingsWithNestedMap{}
	err = myViperEx.Unmarshal(&settings)
	require.NoError(t, err)

	assert.Equal(t, expectedURI, settings.Name)
	assert.Equal(t, expectedURI, settings.NestedMap.Eggs["bob"].SomeValues[1].Value)
	assert.Equal(t, expectedURI, settings.MasterEgg.Name)
}

func TestViperSurgicalUpdate_NestedMap(t *testing.T) {
	configPath := getConfigPath()
	myViper, err := ReadAppsettings(configPath)
	require.NoError(t, err)

	allSettings := myViper.AllSettings()
	t.Log(prettyJSON(allSettings))

	myViperEx, err := New(allSettings, func(ve *ViperEx) error {
		ve.KeyDelimiter = "__"
		return nil
	})
	require.NoError(t, err)

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

	t.Log(prettyJSON(allSettings))

	settings := SettingsWithNestedMap{}
	err = myViperEx.Unmarshal(&settings)
	require.NoError(t, err)
	assert.Equal(t, "bowie", settings.Name)
	assert.Equal(t, int32(1234), settings.NestedMap.Eggs["bob"].Weight)
	assert.Equal(t, "abcd", settings.NestedMap.Eggs["bob"].SomeValues[1].Value)
	assert.Equal(t, "abcd", settings.NestedMap.Eggs["bob"].SomeStrings[1])
	_, ok := allSettings["junk"]
	assert.False(t, ok)

	_, found := myViperEx.Find("name")
	assert.True(t, found)

	_, found = myViperEx.Find("nestedMap__Eggs__junk")
	assert.False(t, found)

	_, found = myViperEx.Find("nestedMap__Eggs__junk__SomeStrings__1__Value")
	assert.False(t, found)
}
func TestViperSurgicalUpdate(t *testing.T) {
	configPath := getConfigPath()
	myViper, err := ReadAppsettings(configPath)
	require.NoError(t, err)

	allSettings := myViper.AllSettings()
	t.Log(prettyJSON(allSettings))

	myViperEx, err := New(allSettings, func(ve *ViperEx) error {
		ve.KeyDelimiter = "__"
		return nil
	})
	require.NoError(t, err)
	envs := map[string]interface{}{
		"nest__MasterEgg__Weight":             1,
		"nest__Eggs__0__Weight":               1234,
		"nest__Eggs__0__Weight__":             1234,
		"nest__Eggs__0__SomeValues__1__Value": "abcd",
		"nest__Eggs__0__SomeStrings__1":       "abcd",
		"nest__Eggs__0__SomeStrings__1__":     "abcd",
		"junk__A":                             "abcd",
		"nest__junk":                          "abcd",
		"nest__tags__0":                       "abcd",
		"nest__tags__1":                       "abcd",
		"somestrings__0":                      "abcd",
	}
	for k, v := range envs {
		myViperEx.UpdateDeepPath(k, v)
	}

	t.Log(prettyJSON(allSettings))

	settings := Settings{}
	err = myViperEx.Unmarshal(&settings)
	require.NoError(t, err)
	assert.Equal(t, "abcd", settings.SomeStrings[0])
	assert.Equal(t, "bob", settings.Name)
	assert.Equal(t, int32(1), settings.Nest.MasterEgg.Weight)
	assert.Equal(t, int32(1234), settings.Nest.Eggs[0].Weight)
	assert.Equal(t, "abcd", settings.Nest.Eggs[0].SomeValues[1].Value)
	assert.Equal(t, "abcd", settings.Nest.Eggs[0].SomeStrings[1])
	assert.Equal(t, "abcd", settings.Nest.Tags[0])
	_, ok := allSettings["junk"]
	assert.False(t, ok)

	_, found := myViperEx.Find("name")
	assert.True(t, found)

	_, found = myViperEx.Find("nest__junk")
	assert.False(t, found)

	var item interface{}

	item, found = myViperEx.Find("nest__Eggs")
	assert.True(t, found)
	assert.NotNil(t, item)

	item, found = myViperEx.Find("nest__Eggs")
	assert.True(t, found)
	assert.NotNil(t, item)

	_, found = myViperEx.Find("nest__Eggs__")
	assert.False(t, found)

	item, found = myViperEx.Find("nest__Eggs__0__SomeStrings__1")
	assert.True(t, found)
	assert.NotNil(t, item)

	_, found = myViperEx.Find("nest__Eggs__0__SomeStrings__1__")
	assert.False(t, found)

	_, found = myViperEx.Find("nest__Eggs__0__Junk__1")
	assert.False(t, found)

	_, found = myViperEx.Find("nest__junk__0__Junk__1")
	assert.False(t, found)

	_, found = myViperEx.Find("nest__junk__0")
	assert.False(t, found)
}

func TestWithDelimiter(t *testing.T) {
	settings := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"value": "deep",
			},
		},
	}
	ve, err := New(settings, WithDelimiter("__"))
	require.NoError(t, err)
	assert.Equal(t, "__", ve.KeyDelimiter)

	val, found := ve.Find("level1__level2__value")
	assert.True(t, found)
	assert.Equal(t, "deep", val)

	// default delimiter should not work with __ paths
	ve2, err := New(settings)
	require.NoError(t, err)
	assert.Equal(t, ".", ve2.KeyDelimiter)

	val, found = ve2.Find("level1.level2.value")
	assert.True(t, found)
	assert.Equal(t, "deep", val)
}

func TestWithEnvPrefix(t *testing.T) {
	settings := map[string]interface{}{
		"name": "original",
	}

	// Normal prefix
	ve, err := New(settings, WithDelimiter("__"), WithEnvPrefix("MYAPP"))
	require.NoError(t, err)
	assert.Equal(t, "MYAPP_", ve.EnvPrefix)

	// Prefix with trailing underscore should not double up
	ve2, err := New(settings, WithDelimiter("__"), WithEnvPrefix("MYAPP_"))
	require.NoError(t, err)
	assert.Equal(t, "MYAPP_", ve2.EnvPrefix)

	// Empty prefix should be ignored
	ve3, err := New(settings, WithDelimiter("__"), WithEnvPrefix(""))
	require.NoError(t, err)
	assert.Equal(t, "", ve3.EnvPrefix)
}

func TestUpdateDeepPath_ReturnValue(t *testing.T) {
	settings := map[string]interface{}{
		"name": "bob",
		"nest": map[string]interface{}{
			"tags": []interface{}{"A", "B"},
		},
	}
	ve, err := New(settings, WithDelimiter("__"))
	require.NoError(t, err)

	// Existing key returns true
	assert.True(t, ve.UpdateDeepPath("name", "alice"))
	val, found := ve.Find("name")
	assert.True(t, found)
	assert.Equal(t, "alice", val)

	// Non-existing key returns false
	assert.False(t, ve.UpdateDeepPath("nonexistent", "value"))

	// Existing array index returns true
	assert.True(t, ve.UpdateDeepPath("nest__tags__0", "C"))
	val, found = ve.Find("nest__tags__0")
	assert.True(t, found)
	assert.Equal(t, "C", val)

	// Out-of-range array index returns false
	assert.False(t, ve.UpdateDeepPath("nest__tags__99", "X"))

	// Trailing delimiter returns false
	assert.False(t, ve.UpdateDeepPath("name__", "value"))
}

func prettyJSON(obj interface{}) string {
	jsonBytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}
