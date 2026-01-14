//go:build tools

// Package main provides tool dependencies for the project.
// This file ensures that tool and test dependencies are tracked in go.mod.
// See: https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md
package main

import (
	_ "github.com/jarcoal/httpmock"
	_ "github.com/stretchr/testify/assert"
	_ "gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
	_ "gotest.tools/gotestsum"
)
