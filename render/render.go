// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import (
	"github.com/xupingao/go-easy-adapt/http"
)

// Render interface is to be implemented by JSON, XML, HTML, YAML and so on.
type Render interface {
	// Render writes data with custom ContentType.
	Render(r http.HTTPResponse) error
	// WriteContentType writes custom ContentType.
	WriteContentType(r http.HTTPResponse)
}

var (
	_ Render     = JSON{}
	_ Render     = IndentedJSON{}
	_ Render     = SecureJSON{}
	_ Render     = JsonpJSON{}
	_ Render     = XML{}
	_ Render     = String{}
	_ Render     = Redirect{}
	_ Render     = Data{}
	_ Render     = HTML{}
	_ HTMLRender = HTMLDebug{}
	_ HTMLRender = HTMLProduction{}
	_ Render     = YAML{}
	_ Render     = Reader{}
	_ Render     = AsciiJSON{}
	_ Render     = ProtoBuf{}
)

func writeContentType(w http.HTTPResponse, value []string) {
	header := w.Header()
	if val := header.Get(http.HeaderContentType); len(val) == 0 {
		header.Set(http.HeaderContentType, value[0])
	}
}
