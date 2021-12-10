// Copyright 2017 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import  "github.com/xupingao/go-easy-adapt/http"
type queryBinding struct{}

func (queryBinding) Name() string {
	return "query"
}

func (queryBinding) Bind(req http.HTTPRequest, obj interface{}) error {
	values := req.URL().Query().All()
	if err := mapForm(obj, values); err != nil {
		return err
	}
	return validate(obj)
}
