// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import (
	"github.com/xupingao/go-easy-adapt/http"

	"gopkg.in/yaml.v2"
)

// YAML contains the given interface object.
type YAML struct {
	Data interface{}
}

var yamlContentType = []string{"application/x-yaml; charset=utf-8"}

// Render (YAML) marshals the given interface object and writes data with custom ContentType.
func (r YAML) Render(w http.HTTPResponse) error {
	r.WriteContentType(w)

	bytes, err := yaml.Marshal(r.Data)
	if err != nil {
		return err
	}

	_, err = w.Write(bytes)
	return err
}

// WriteContentType (YAML) writes YAML ContentType for response.
func (r YAML) WriteContentType(w http.HTTPResponse) {
	writeContentType(w, yamlContentType)
}
