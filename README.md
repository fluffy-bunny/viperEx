# viperEx  

Filling the gaps from the awesome spf13/viper project

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
