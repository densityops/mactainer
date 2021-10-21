/*
Copyright 2019 Red Hat, Inc.
Copyright 2021 DensityOps All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package host

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"
	"strings"
	"time"

	crcerrors "github.com/code-ready/crc/pkg/crc/errors"
	log "github.com/code-ready/crc/pkg/crc/logging"
	"github.com/densityops/machine/libmachine/drivers"
	"github.com/densityops/machine/libmachine/state"
)

// ConfigVersion dictates which version of the config.json format is
// used. It needs to be bumped if there is a breaking change, and
// therefore migration, introduced to the config file format.
const Version = 3

type Host struct {
	ConfigVersion int
	Driver        drivers.Driver
	DriverName    string
	DriverPath    string
	Name          string
	RawDriver     []byte `json:"-"`
	PublicKey     string
	PrivateKey    string
}

type Metadata struct {
	ConfigVersion int
}

func (h *Host) runActionForState(action func() error, desiredState state.State) error {
	if err := MachineInState(h.Driver, desiredState)(); err == nil {
		return fmt.Errorf("machine is already %s", strings.ToLower(desiredState.String()))
	}

	if err := action(); err != nil {
		return err
	}

	return crcerrors.Retry(context.Background(), 3*time.Minute, MachineInState(h.Driver, desiredState), 3*time.Second)
}

func (h *Host) Stop() error {
	log.Debugf("Stopping %q...", h.Name)
	if err := h.runActionForState(h.Driver.Stop, state.Stopped); err != nil {
		return err
	}

	log.Debugf("Machine %q was stopped.", h.Name)
	return nil
}

func (h *Host) Kill() error {
	log.Debugf("Killing %q...", h.Name)
	if err := h.runActionForState(h.Driver.Kill, state.Stopped); err != nil {
		return err
	}

	log.Debugf("Machine %q was killed.", h.Name)
	return nil
}

func (h *Host) UpdateConfig(rawConfig []byte) error {
	err := h.Driver.UpdateConfigRaw(rawConfig)
	if err != nil {
		var e rpc.ServerError
		if errors.As(err, &e) && err.Error() == "Not Implemented" {
			err = drivers.ErrNotImplemented
		}
		return err
	}
	h.RawDriver = rawConfig

	return nil
}

func MachineInState(d drivers.Driver, desiredState state.State) func() error {
	return func() error {
		currentState, err := d.GetState()
		if err != nil {
			return err
		}
		if currentState == desiredState {
			return nil
		}
		return &crcerrors.RetriableError{
			Err: fmt.Errorf("expected machine state %s, got %s", desiredState.String(), currentState.String()),
		}
	}
}
