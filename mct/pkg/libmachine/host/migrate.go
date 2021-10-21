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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/densityops/mactainer/mct/pkg/drivers/none"
)

var (
	errUnexpectedConfigVersion = errors.New("unexpected config version")
)

type RawDataDriver struct {
	*none.Driver
	Data []byte // passed directly back when invoking json.Marshal on this type
}

func (r *RawDataDriver) MarshalJSON() ([]byte, error) {
	return r.Data, nil
}

func (r *RawDataDriver) UnmarshalJSON(data []byte) error {
	r.Data = data
	return nil
}

func (r *RawDataDriver) UpdateConfigRaw(rawData []byte) error {
	return r.UnmarshalJSON(rawData)
}

func MigrateHost(name string, data []byte) (*Host, error) {
	var hostMetadata Metadata
	if err := json.Unmarshal(data, &hostMetadata); err != nil {
		return nil, err
	}

	if hostMetadata.ConfigVersion != Version {
		return nil, errUnexpectedConfigVersion
	}

	driver := &RawDataDriver{none.NewDriver(name, ""), nil}
	h := Host{
		Name:   name,
		Driver: driver,
	}
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("Error unmarshalling most recent host version: %s", err)
	}
	h.RawDriver = driver.Data
	return &h, nil
}
