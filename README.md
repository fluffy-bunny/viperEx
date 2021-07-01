# viperEx  

Filling the gaps from the awesome spf13/viper project.  
There is an [spf13/viper--issue](https://github.com/spf13/viper/issues/1140) that reference this project.  

## The Gaps

Asp.Net core allows for deep updating a configuration via an opinionated pathing ENV variable.  

Given a configuration structs below;

```go
type Nest struct {
  Name       string
  CountInt   int
  CountInt16 int16
  Eggs       []Egg
}

type NestedMap struct {
  Eggs map[string]Egg
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

type SettingsWithNestedMap struct {
  Name      string
  NestedMap *NestedMap
}

```

I would like to surgically update a value deep in the tree.  
I want it to dig down and enter into the right array or map object and keep going.  

## Arrays  

Here I would like to change value of ```Value```, which is in the 2nd object in the ```SomeValues``` array, which is in an ```Egg``` object which is in the 2nd object in the ```Eggs``` array, which is inside or the ```nest``` object.  Phewww!  

```bash
nest__Eggs__1__SomeValues__1__Value=3
```  

Here it accounts for arrays, where Eggs is an array.  The Egg stuct also contains an array

```go
  os.Setenv("APPLICATION_ENVIRONMENT", "Test")
  os.Setenv("nest__Eggs__1__Weight", "5555")
  os.Setenv("nest__Eggs__1__SomeValues__1__Value", "Heidi") // update an item in a struct
  os.Setenv("nest__Eggs__1__SomeStrings__1", "Zep") // SomeStrings is a []string, so this is the convention for directly modifying a primitive in an array
```

```go
  allSettings := myViper.AllSettings() // normal viper stuff

  myViperEx, err := New(allSettings, func(ve *ViperEx) error {
    ve.KeyDelimiter = "__"
    return nil
  })
  myViperEx.UpdateFromEnv()

  // or individually
  myViperEx.UpdateDeepPath("nest__Eggs__0__Weight", 1234)
  myViperEx.UpdateDeepPath("nest__Eggs__0__SomeValues__1__Value", "abcd")
  myViperEx.UpdateDeepPath("nest__Eggs__0__SomeStrings__1", "abcd")

  // since we took ownership of the all settings we need to use our own Unmarshal
  err = myViperEx.Unmarshal(&settings)
```

## Maps  

Maps are very similar to arrys, except that a ```string``` is used instead of a ```num``` for pathing.

### Array pathing  

```bash
nest__Eggs__1__SomeValues__1__Value=3
```  

vs.

### Map pathing  

```bash
nest__Eggs__bob__SomeValues__1__Value=3
```  
