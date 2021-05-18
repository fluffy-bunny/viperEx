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

type Nest struct {
	Name       string
	CountInt   int
	CountInt16 int16
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
	os.Setenv("APPLICATION_ENVIRONMENT", "Test")
	os.Setenv("nest__Eggs__1__Weight", "5555")
	os.Setenv("nest__Eggs__1__SomeValues__1__Value", "Heidi")
	os.Setenv("nest__Eggs__1__SomeStrings__1__", "Zep")
	chdirToTestFolder()
}

const keyDelim = "__"

func ReadAppsettings(rootPath string) (*viper.Viper, error) {
	var err error
	environment := os.Getenv("APPLICATION_ENVIRONMENT")
	myViper := viper.NewWithOptions(viper.KeyDelimiter("__"))
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

	myViperEx := New("__")
	myViperEx.UpdateFromEnv(allSettings)

	settings := Settings{}
	err = myViper.Unmarshal(&settings)
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "Heidi", settings.Nest.Eggs[1].SomeValues[1].Value)
	assert.Equal(t, "Zep", settings.Nest.Eggs[1].SomeStrings[1])
	assert.Equal(t, int32(5555), settings.Nest.Eggs[1].Weight)
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

	myViperEx := New("__")
	myViperEx.SurgicalUpdate("nest__Eggs__0__Weight", 1234, allSettings)
	myViperEx.SurgicalUpdate("nest__Eggs__0__SomeValues__1__Value", "abcd", allSettings)
	myViperEx.SurgicalUpdate("nest__Eggs__0__SomeStrings__1__", "abcd", allSettings)
	myViperEx.SurgicalUpdate("junk__A", "abcd", allSettings)
	myViperEx.SurgicalUpdate("nest__junk", "abcd", allSettings)

	fmt.Println(PrettyJSON(allSettings))

	settings := Settings{}
	err = myViper.Unmarshal(&settings)
	assert.NoError(t, err)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "bob", settings.Name)
	assert.Equal(t, int32(1234), settings.Nest.Eggs[0].Weight)
	assert.Equal(t, "abcd", settings.Nest.Eggs[0].SomeValues[1].Value)
	assert.Equal(t, "abcd", settings.Nest.Eggs[0].SomeStrings[1])
	_, ok := allSettings["junk"]
	assert.False(t, ok)

	nestJunk := myViperEx.Find("nest__junk", allSettings)
	assert.Nil(t, nestJunk)

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
