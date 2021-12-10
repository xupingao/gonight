// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import (
	"fmt"
	"github.com/xupingao/go-easy-adapt/http"
)

// Redirect contains the http request reference and redirects status code and location.
type Redirect struct {
	Code     int
	Request  http.HTTPRequest
	Location string
}

// Render (Redirect) redirects the http request to new location and writes redirect response.
func (r Redirect) Render(w http.HTTPResponse) error {
	if (r.Code < http.StatusMultipleChoices || r.Code > http.StatusPermanentRedirect) && r.Code != http.StatusCreated {
		panic(fmt.Sprintf("Cannot redirect with status code %d", r.Code))
	}
	// todo
	//http.Redirect(w, r.Request, r.Location, r.Code)
	return nil
}

// WriteContentType (Redirect) don't write any ContentType.
func (r Redirect) WriteContentType(http.HTTPResponse) {}
