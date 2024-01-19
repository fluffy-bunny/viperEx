package viperEx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jinzhu/copier"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

var (
	ConfigDefaultJSON = []byte(`
{
	"nest": {}
}
`)
	stemEgg = `{
	"name": "in-environment",
	"weight": 0,
	"somestrings": [
		"in-environment","in-environment", "in-environment"
	],
	"somevalues": [{
		"value": "in-environment"
	}, {
		"value": "in-environment"
	}]
}`
)

type (
	DynamicNest struct {
		Eggs []*Egg `json:"eggs"`
	}
	DynamicConfig struct {
		Nest *DynamicNest `json:"nest"`
	}
)

func TestViperExEnvUpdate_dynamic_nest(t *testing.T) {
	const numEggs = 3
	var egg = &Egg{}
	err := json.Unmarshal([]byte(stemEgg), egg)
	require.NoError(t, err)
	var dynamicConfig = &DynamicConfig{}
	err = json.Unmarshal([]byte(ConfigDefaultJSON), dynamicConfig)
	require.NoError(t, err)
	for i := 0; i < numEggs; i++ {
		newEgg := &Egg{}
		copier.Copy(&newEgg, egg)
		dynamicConfig.Nest.Eggs = append(dynamicConfig.Nest.Eggs, newEgg)
	}
	finalJSON, err := json.Marshal(dynamicConfig)
	require.NoError(t, err)

	myViper := viper.NewWithOptions(viper.KeyDelimiter(keyDelim))
	myViper.SetConfigType("json")
	myViper.AutomaticEnv()
	myViper.ReadConfig(bytes.NewBuffer(finalJSON))

	allSettings := myViper.AllSettings()
	fmt.Println(PrettyJSON(allSettings))

	myViperEx, err := New(allSettings, func(ve *ViperEx) error {
		ve.KeyDelimiter = keyDelim
		return nil
	})

	envs := map[string]string{
		"nest__eggs__0__name":                 "name0",
		"nest__eggs__0__weight":               "1",
		"nest__eggs__0__somestrings__0":       "name0_somestring0",
		"nest__eggs__0__somestrings__1":       "name0_somestring1",
		"nest__eggs__0__somestrings__2":       "name0_somestring2",
		"nest__eggs__0__SomeValues__0__Value": "name0_somevalue0",
		"nest__eggs__0__SomeValues__1__Value": "name0_somevalue1",

		"nest__eggs__1__name":                 "name1",
		"nest__eggs__1__weight":               "2",
		"nest__eggs__1__somestrings__0":       "name1_somestring0",
		"nest__eggs__1__somestrings__1":       "name1_somestring1",
		"nest__eggs__1__somestrings__2":       "name1_somestring2",
		"nest__eggs__1__SomeValues__0__Value": "name1_somevalue0",
		"nest__eggs__1__SomeValues__1__Value": "name1_somevalue1",

		"nest__eggs__2__name":                 "name2",
		"nest__eggs__2__weight":               "3",
		"nest__eggs__2__somestrings__0":       "name2_somestring0",
		"nest__eggs__2__somestrings__1":       "name2_somestring1",
		"nest__eggs__2__somestrings__2":       "name2_somestring2",
		"nest__eggs__2__SomeValues__0__Value": "name2_somevalue0",
		"nest__eggs__2__SomeValues__1__Value": "name2_somevalue1",
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}
	err = myViperEx.UpdateFromEnv()
	require.NoError(t, err)

	settings := &DynamicConfig{}
	err = myViperEx.Unmarshal(&settings)
	fmt.Println(PrettyJSON(allSettings))

	for idx, egg := range settings.Nest.Eggs {
		require.Equal(t, fmt.Sprintf("name%d", idx), egg.Name, "egg name")
		require.Equal(t, int32(idx+1), egg.Weight, "egg weight")
		for ii, ss := range egg.SomeStrings {
			require.Equal(t, fmt.Sprintf("name%d_somestring%d", idx, ii), ss, "egg somestring")
		}
		for ii, sv := range egg.SomeValues {
			require.Equal(t, fmt.Sprintf("name%d_somevalue%d", idx, ii), sv.Value, "egg somevalue")
		}
	}
}
