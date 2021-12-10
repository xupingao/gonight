// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import (
	"github.com/xupingao/go-easy-adapt/http"
)

const defaultMemory = 32 << 20

type formBinding struct{}
type formPostBinding struct{}
//type formMultipartBinding struct{}

func (formBinding) Name() string {
	return "form"
}

func (formBinding) Bind(req http.HTTPRequest, obj interface{}) error {
	//if err := req.ParseForm(); err != nil {
	//	return err
	//}
	//if err := req.ParseMultipartForm(defaultMemory); err != nil {
	//	if err != http.ErrNotMultipart {
	//		return err
	//	}
	//}
	if err := mapForm(obj, req.Form().All()); err != nil {
		return err
	}
	return validate(obj)
}

func (formPostBinding) Name() string {
	return "form-urlencoded"
}

func (formPostBinding) Bind(req http.HTTPRequest, obj interface{}) error {

	if err := mapForm(obj, req.PostForm().All()); err != nil {
		return err
	}
	return validate(obj)
}

//func (formMultipartBinding) Name() string {
//	return "multipart/form-data"
//}
//
//func (formMultipartBinding) Bind(req http.HTTPRequest, obj interface{}) error {
//	//if err := req.ParseMultipartForm(defaultMemory); err != nil {
//	//	return err
//	//}
//	if err := mappingByPtr(obj, (*multipartRequest)(req), "form"); err != nil {
//		return err
//	}
//
//	return validate(obj)
//}
