module github.com/kalo-build/plugin-morphediff-psql

go 1.24.0

toolchain go1.24.2

require (
	github.com/gertd/go-pluralize v0.2.1
	github.com/kalo-build/go-util v0.0.0-20250329083327-00e97aeff9b7
	github.com/kalo-build/kalo-sdk-go v0.0.0
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/kalo-build/kalo-sdk-go => ../kalo-sdk-go

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gobeam/stringy v0.0.7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)
