// Copyright (c) 2022 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The tsnet-funnel server demonstrates how to use Funnel from tsnet. It also
// demonstrates how to have a separate HTTP listener for public vs private traffic.
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"

	"tailscale.com/tsnet"
)

var (
	addr = flag.String("addr", ":443", "address to listen on")
	hostname = flag.String("hostname", "tsnet-funnel", "hostname to use on your tailnet")
)

func main() {
	flag.Parse()
	s := new(tsnet.Server)
	s.Hostname = *hostname
	defer s.Close()

	publicLis, err := s.ExposeHTTPS()
	if err != nil {
		log.Fatal(err)
	}

	ln, err := s.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	lc, err := s.LocalClient()
	if err != nil {
		log.Fatal(err)
	}

	ln = tls.NewListener(ln, &tls.Config{
		GetCertificate: lc.GetCertificate,
	})

	go log.Fatal(http.Serve(publicLis, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<html><body><h1>Hello, internet!</h1>")
		fmt.Fprintln(w, "<p>You are connected over the internet!</p></html>")
	})))
	log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<html><body><h1>Hello, tailnet!</h1>")
		fmt.Fprintln(w, "<p>You are connected over the tailnet!</p></html>")
	})))
}
