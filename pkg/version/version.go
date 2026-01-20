/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package version provides version information for the operator.
package version

import (
	"encoding/json"
	"net/http"
)

// These variables are set at build time via -ldflags.
var (
	// Version is the operator version.
	Version = "0.1.0"
	// Commit is the git commit hash.
	Commit = "unknown"
	// BuildTime is the build timestamp.
	BuildTime = "unknown"
)

// Info contains version information.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"buildTime"`
}

// Get returns the current version info.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}
}

// Handler returns an http.HandlerFunc that serves version info as JSON.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Get())
	}
}
