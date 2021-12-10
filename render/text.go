// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import (
	"fmt"
	"github.com/xupingao/go-easy-adapt/http"
	"io"
)

// String contains the given interface object slice and its format.
type String struct {
	Format string
	Data   []interface{}
}

var plainContentType = []string{"text/plain; charset=utf-8"}

// Render (String) writes data with custom ContentType.
func (r String) Render(w http.HTTPResponse) error {
	return WriteString(w, r.Format, r.Data)
}

// WriteContentType (String) writes Plain ContentType.
func (r String) WriteContentType(w http.HTTPResponse) {
	writeContentType(w, plainContentType)
}

// WriteString writes data according to its format and write custom ContentType.
func WriteString(w http.HTTPResponse, format string, data []interface{}) (err error) {
	writeContentType(w, plainContentType)
	if len(data) > 0 {
		_, err = fmt.Fprintf(w, format, data...)
		return
	}
	_, err = io.WriteString(w, format)
	return
}
