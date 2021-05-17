# viperEx  

Filling the gaps from the awesome spf13/viper project

# The Gaps  
Asp.Net core allows for deep updating a configuration via an opinionated pathing ENV variable.  

Given a configuration structs below;
```go
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
```
I would like to surgically update a value deep in the tree.  
I want it to dig down and enter into the right array object and keep going.  

Here I would like to change value of ```Value```, which is in the 2nd object in the ```SomeValues``` array, which is in an ```Egg``` object which is in the 2nd object in the ```Eggs``` array, which is inside or the ```nest``` object.  Phewww!  

```
nest__Eggs__1__SomeValues__1__Value=3
```  

Here it accounts for arrays, where Eggs is an array.  The Egg stuct also contains an array



I don't use AutomaticEnv from viper and instead do this at the end of my configuration build.  

```go
  os.Setenv("APPLICATION_ENVIRONMENT", "Test")
  os.Setenv("nest__Eggs__1__Weight", "5555")
  os.Setenv("nest__Eggs__1__SomeValues__1__Value", "Heidi") // update an item in a struct
  os.Setenv("nest__Eggs__1__SomeStrings__1__", "Zep") // SomeStrings is a []string, so this is the convention for directly modifying a primitive in an array
```

```go
  allSettings := myViper.AllSettings() // normal viper stuff

  myViperEx := viperEx.New("__")
  myViperEx.UpdateFromEnv(allSettings)

  // or individually
  myViperEx.SurgicalUpdate("nest__Eggs__0__Weight", 1234, allSettings)
  myViperEx.SurgicalUpdate("nest__Eggs__0__SomeValues__1__Value", "abcd", allSettings)
  myViperEx.SurgicalUpdate("nest__Eggs__0__SomeStrings__1__", "abcd", allSettings)

  // you can use vipers unmarshal still
  // The original allSettings was pulled from viper and modified by viperEx
  err = myViper.Unmarshal(&settings)
```
