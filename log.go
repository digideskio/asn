// Copyright 2014 Apptimist, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !nolog

package main

var Log = &Logger{}

func init() { Log.Init() }
