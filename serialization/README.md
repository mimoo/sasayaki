Should I include the building directive in a go:generate ?

probably not, because it will require people to have protobuff before getting sasayaki


```
protoc --go_out=. messages.proto
```

^ this should be run by me, and then pushed.