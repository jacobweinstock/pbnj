// Copyright 2020 - 2020, Packethost, Inc and contributors
// SPDX-License-Identifier: Apache-2.0

package racadm

import (
	"fmt"

	"github.com/jacobweinstock/pbnj/interfaces/bmc"
	"github.com/pkg/errors"
)

var lanIPSources = map[bmc.IPSource]string{
	bmc.IPFromDHCP: "0",
	bmc.StaticIP:   "1",
}

// idrac.ipv4

// SetIPSource sets ip configuration method
func (s *Shell) SetIPSource(source bmc.IPSource) error {
	arg, ok := lanIPSources[source]
	if !ok {
		return errors.New(fmt.Sprintf("ip source %q not supported by racadm driver", source))
	}

	return s.Run("racadm set idrac.ipv4.dhcpenable" + " " + arg)
}
