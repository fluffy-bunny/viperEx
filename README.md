# viperEx  

Filling the gaps from the awesome spf13/viper project.  
There is an [spf13/viper--issue](https://github.com/spf13/viper/issues/1140) that references this project.  

## The Gaps

ASP.NET Core allows for deep updating a configuration via an opinionated pathing ENV variable.  
ViperEx brings this capability to Go's [spf13/viper](https://github.com/spf13/viper).

Given the configuration structs below:

```go
type Nest struct {
  Name       string
  CountInt   int
  CountInt16 int16
  MasterEgg  Egg
  Eggs       []Egg
  Tags       []string
}

type NestedMap struct {
  Eggs map[string]Egg
}

type ValueContainer struct {
  Value interface{}
}

type Egg struct {
  Weight      int32
  SomeValues  []ValueContainer
  SomeStrings []string
  Name        string
}

type Settings struct {
  Name        string
  Nest        *Nest
  SomeStrings []string
}

type SettingsWithNestedMap struct {
  Name      string
  NestedMap *NestedMap
  MasterEgg Egg
}
```

I would like to surgically update a value deep in the tree.  
I want it to dig down and enter into the right array or map object and keep going.  

## Quick Start

```go
allSettings := myViper.AllSettings() // normal viper stuff

// Create a ViperEx instance (does not modify the original map)
myViperEx, err := New(allSettings, WithDelimiter("__"))

// Bulk update from environment variables
myViperEx.UpdateFromEnv()

// Or update individual paths (returns true if path was found)
ok := myViperEx.UpdateDeepPath("nest__Eggs__0__Weight", 1234)

// Find a value (returns value and whether it was found)
val, found := myViperEx.Find("nest__Eggs__0__Weight")

// Unmarshal into your struct
err = myViperEx.Unmarshal(&settings)
```

## Options

```go
// Set a custom key delimiter (default is ".")
myViperEx, err := New(allSettings, WithDelimiter("__"))

// Filter env vars by prefix (e.g. only MYAPP_some__key)
myViperEx, err := New(allSettings, WithDelimiter("__"), WithEnvPrefix("MYAPP"))
```

## Arrays  

Here I would like to change the value of `Value`, which is in the 2nd object in the `SomeValues` array, which is in an `Egg` object which is in the 2nd object in the `Eggs` array, which is inside the `nest` object.  Phew!  

```bash
nest__Eggs__1__SomeValues__1__Value=3
```  

Here it accounts for arrays, where Eggs is an array.  The Egg struct also contains an array.

```go
  t.Setenv("nest__Eggs__1__Weight", "5555")
  t.Setenv("nest__Eggs__1__SomeValues__1__Value", "Heidi")
  t.Setenv("nest__Eggs__1__SomeStrings__1", "Zep")
```

```go
  allSettings := myViper.AllSettings()

  myViperEx, err := New(allSettings, WithDelimiter("__"))
  myViperEx.UpdateFromEnv()

  // or individually (returns true if path exists)
  myViperEx.UpdateDeepPath("nest__Eggs__0__Weight", 1234)
  myViperEx.UpdateDeepPath("nest__Eggs__0__SomeValues__1__Value", "abcd")
  myViperEx.UpdateDeepPath("nest__Eggs__0__SomeStrings__1", "abcd")

  // since we took ownership of the settings we need to use our own Unmarshal
  err = myViperEx.Unmarshal(&settings)
```

## Maps  

Maps are very similar to arrays, except that a `string` is used instead of a `num` for pathing.

### Array pathing  

```bash
nest__Eggs__1__SomeValues__1__Value=3
```  

vs.

### Map pathing  

```bash
nestedMap__Eggs__bob__SomeValues__1__Value=3
```  

## Custom Duration Unmarshalling

ViperEx supports unmarshalling human-readable duration strings (e.g. `"5s"`, `"2m30s"`) into custom types that implement `encoding.TextUnmarshaler`, as well as the standard `time.Duration`.

```go
type MyDuration time.Duration

func (d *MyDuration) UnmarshalText(text []byte) error {
  parsed, err := time.ParseDuration(string(text))
  if err != nil {
    return err
  }
  *d = MyDuration(parsed)
  return nil
}
```

## Limitations

- **Arrays-of-arrays are not supported.** Deep-path traversal handles maps containing arrays and arrays containing maps, but nested arrays (e.g. `[][]interface{}`) will not be traversed.
