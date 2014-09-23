// Copyright 2014 Apptimist, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
	"os"
	"testing"
)

func TestConfig(t *testing.T) {
	for _, tt := range []string{
		"testdata/adm.yaml",
	} {
		c, err := New(tt)
		if err != nil {
			t.Error(err)
			continue
		}
		fmt.Print("# ", tt, "\n", c, "\n")
		c.Clean()
		c = nil
	}
}

/*
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
*/

func TestNegative(t *testing.T) {
	for _, tt := range []struct {
		expect string
		config string
	}{
		{"no name", Inline + `
# name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no keys", Inline + `
name: foobar
dir: foobar-adm.asn
lat: 37.774929
lon: -122.419415
# keys:
#   admin:
#     pub:
#       encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
#       auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
#     sec:
#       encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
#       auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
#   server:
#     pub:
#       encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
#       auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
#   nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no admin keys", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
#   admin:
#     pub:
#       encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
#       auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
#     sec:
#       encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
#       auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no admin public keys", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
#     pub:
#       encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
#       auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no admin public encr key", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
#       encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- unix///foobar.sock
`},
		{"no admin public auth key", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
#       auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no admin secret keys", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
#     sec:
#       encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
#       auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no admin secret encr key", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
#       encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no admin secret auth key", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
#       auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no server keys", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
#   server:
#     pub:
#       encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
#       auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no server public keys", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
#     pub:
#       encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
#       auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no server public encr key", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
#       encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no server public auth key", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
#       auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"no nonce", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
#   nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
server:
- name: Local
  url: unix:///var/run/asn/siren.sock
  lat: 181
  lon: 0
`},
		{"not servers", Inline + `
name: foobar
lat: 37.774929
lon: -122.419415
keys:
  admin:
    pub:
      encr: 5fb2d5d9552c47f02d4cfc1f3938abd4c5f685b050501e53f6bf545c05982e33
      auth: 9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
    sec:
      encr: f6ce8a1025b3537e3a82ab5461fa7a2db51a2729abe66cdce82b54a573de011d
      auth: 60eabf950dc926735d086f419b2571de6e95c4e1d1efe179590b1acc8ffee39c9d30799789fb96a2d71855168d8573d2ce6f367e6a0ef7da7bcee72ab31dcc13
  server:
    pub:
      encr: 45236651d82a9f5b098038e0f23186cd53b7dc487a99af68c41c9a3b0e7f6d00
      auth: b37cb08f941a01d299d6609e471e7b9e43d456a5920cb9151b9ed698537882c2
  nonce: a7967df6f832bbdfc30c301ff22f3929cd4b900610541d39
# server:
# - name: Local
#   url: unix:///var/run/asn/siren.sock
#   lat: 181
#   lon: 0
`},
	} {
		c, err := New(tt.config)
		if err == nil {
			t.Error("expected error:", tt.expect)
		} else {
			fmt.Fprintln(os.Stdout, "PASS:", err)
		}
		c.Clean()
		c = nil
	}
}