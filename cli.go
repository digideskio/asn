// Copyright 2014-2015 Apptimist, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !nocli

package main

import (
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rocky/go-gnureadline"
)

func (adm *Adm) CLI() (err error) {
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		return
	}
	home := os.Getenv("HOME")
	history := filepath.Join(home, ".asnadm_history")
	defer gnureadline.WriteHistory(history)
	rc := filepath.Join(home, ".asnadmrc")
	if _, err = os.Stat(rc); err == nil {
		if err = gnureadline.ReadInitFile(rc); err != nil {
			return
		}
	}
	if _, err = os.Stat(history); err == nil {
		gnureadline.ReadHistory(history)
	} else {
		err = nil
	}
	gnureadline.StifleHistory(32)
	done := make(Done, 1)
	defer close(done)
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	defer signal.Stop(winch)
	quit := false
	line := ""
	go func() {
		for {
			var rlerr error
			line = ""
			line, rlerr = gnureadline.Readline("asn# ", true)
			if rlerr != nil || quit {
				break
			}
			err = adm.cmdLine(line)
			if err != nil {
				println(err.Error())
				if err == io.EOF {
					quit = true
				} else {
					err = nil
				}
			}
			if quit {
				break
			}
		}
		done <- nil
	}()
	defer gnureadline.Rl_reset_terminal("")
	for !quit {
		select {
		case <-done:
			return
		case pdu, opened := <-adm.clich:
			if !opened {
				quit = true
				proc.Signal(os.Kill)
				<-done
			} else if line == "" {
				println()
				adm.ObjDump(pdu)
				gnureadline.Rl_resize_terminal()
			} else {
				adm.ObjDump(pdu)
			}
		case <-winch:
			gnureadline.Rl_resize_terminal()
		}
	}
	return
}
