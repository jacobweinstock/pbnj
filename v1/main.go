// Copyright 2020 - 2020, Packethost, Inc and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"os"
	"time"

	"github.com/jacobweinstock/pbnj/api"
	"github.com/jacobweinstock/pbnj/drivers/ipmitool"
	"github.com/jacobweinstock/pbnj/drivers/racadm"
	"github.com/jacobweinstock/pbnj/interfaces/power"
	"github.com/jacobweinstock/pbnj/log"
	"github.com/jacobweinstock/pbnj/reqid"
)

var (
	GitRev     = "unknown (use make)"
	listenAddr = ":9090"
)

func init() {
	if nomadPort, ok := os.LookupEnv("NOMAD_PORT_internal_http"); ok {
		listenAddr = ":" + nomadPort
	}

	flag.StringVar(&listenAddr, "listen-addr", listenAddr, "IP and port to listen on for HTTP")
}

func cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		power.CleanupTasks(3 * time.Minute)
	}
}

func main() {
	flag.Parse()

	logger, sync, err := log.Init("github.com/jacobweinstock/pbnj")
	if err != nil {
		panic(err)
	}
	defer sync()

	api.SetupLogging(logger)
	ipmitool.SetupLogging(logger)
	power.SetupLogging(logger)
	racadm.SetupLogging(logger)
	reqid.SetupLogging(logger)
	go cleanup()

	logger = logger.Package("main")
	if err := api.Serve(listenAddr, GitRev); err != nil {
		logger.Fatalw("error serving api", "error", err)
	}
}
