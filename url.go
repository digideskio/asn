// Copyright 2014-2015 Apptimist, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "net/url"

type URL url.URL

func NewURL(s string) (*URL, error) {
	p, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	return (*URL)(p), nil
}

func (u *URL) GetYAML() (string, interface{}) {
	return "", u.String()
}

func (u *URL) SetYAML(t string, v interface{}) bool {
	if s, ok := v.(string); ok {
		if p, err := url.Parse(s); err == nil {
			*u = URL(*p)
			return true
		}
	}
	return false
}

func (u *URL) String() string {
	return ((*url.URL)(u)).String()
}
