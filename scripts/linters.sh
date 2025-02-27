#!/bin/bash

# suspicious constructs
go vet ./database
go vet ./event_processor
go vet ./ui
go vet ./main.go

# code style checker
golint ./database
golint ./event_processor
golint ./ui
golint ./main.go

# statick analizer
staticcheck -checks=all ./database
staticcheck -checks=all ./event_processor
staticcheck -checks=all ./ui
staticcheck -checks=all ./main.go

# TODO: pprof, race detector, test coverage, etc.
