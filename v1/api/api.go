// Copyright 2020 - 2020, Packethost, Inc and contributors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jacobweinstock/pbnj/api/redfish"
	"github.com/jacobweinstock/pbnj/evlog"
	"github.com/jacobweinstock/pbnj/interfaces/bmc"
	"github.com/jacobweinstock/pbnj/interfaces/boot"
	"github.com/jacobweinstock/pbnj/interfaces/power"
	"github.com/jacobweinstock/pbnj/log"
	"github.com/packethost/pkg/env"
	"github.com/pkg/errors"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

var gitRev string

var (
	packetEnv = env.Get("PACKET_ENV", "UNKNOWN")
	logger    log.Logger
	elog      *evlog.Log
)

func SetupLogging(l log.Logger) {
	bmc.SetupLogging(l)
	boot.SetupLogging(l)
	power.SetupLogging(l)

	logger = l.Package("api")
	elog = evlog.New(logger)
}

// Serve serves http api
func Serve(addr, rev string) error {
	gitRev = rev
	r := gin.New()
	r.Use(logging, jsonErrors, recovery)

	// no auth required
	r.GET("/healthcheck", healthcheck)
	r.GET("/_packet/healthcheck", healthcheck)
	r.GET("/_packet/version", healthcheck)

	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	// everything from now on must have valid auth
	r.Use(authorize)
	r.GET("/tasks/:id", taskStatus)
	r.GET("/", ping)

	device := r.Group("/devices/:ip")

	// Power Control
	device.GET("/power", powerStatus)
	device.POST("/power", powerAction)

	// Boot Options
	device.PATCH("/boot", updateBootOptions)

	// BMC
	device.POST("/bmc", bmcAction)

	// Management LAN Options
	device.PATCH("/ipmi-lan", updateLANConfig)

	// Redfish proxy
	device.Any("/redfish/*redfish", redfish.Proxy)

	return errors.Wrap(r.Run(addr), "serving http api")
}

func ping(c *gin.Context) {
	c.Writer.WriteHeader(http.StatusNoContent)
}

func healthcheck(c *gin.Context) {
	c.JSON(http.StatusOK, struct {
		Name string `json:"name"`
		Rev  string `json:"rev"`
		Env  string `json:"env"`
	}{
		Name: "pbnj",
		Rev:  gitRev,
		Env:  packetEnv,
	})
}

// logging is a gin middleware that logs http method, path and client id.
// logging also adds a Header to the gin context... I don't know why/what it's used for since all logs get it from Tx.
func logging(c *gin.Context) {
	var fields []interface{}
	tx := elog.TxFromContext(c)

	log := true
	path := c.Request.RequestURI
	if strings.HasPrefix(path, "/_packet") || path == "/metrics" {
		log = false
	}
	if log {
		start := time.Now()
		method := c.Request.Method
		client := c.ClientIP()

		fields = []interface{}{
			"method", method, "path", path, "client", client,
		}
		tx.Info("request_start", fields...)

		defer func() {
			duration := time.Since(start)
			status := c.Writer.Status()
			fields = append(fields, "duration", duration, "status", status)
			tx.Notice("request_end", fields...)
		}()
	}

	c.Header("X-Request-ID", tx.ID)
	c.Next() // Process the request
}

// recovery is a gin middleware that catches any panics, turns it into an error (with stack frame) and logs it.
func recovery(c *gin.Context) {
	tx := elog.TxFromContext(c)
	defer func() {
		iface := recover()
		if iface == nil {
			return
		}

		var err error
		switch iface := iface.(type) {
		case error:
			err = iface
		default:
			err = errors.Errorf("%v", iface)
		}
		err = errors.WithMessage(err, "panic recovered")

		tx.Panic("request_panic", "error", err)
		internalServerError(c)
	}()
	c.Next()
}
