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
package libmachine

import (
	"github.com/densityops/mactainer/mct/pkg/libmachine/host"
	"github.com/densityops/mactainer/mct/pkg/libmachine/ssh"
)

func (api *Client) NewHost(driverName string, driverPath string, rawDriver []byte) (*host.Host, error) {
	driver, err := api.clientDriverFactory.NewRPCClientDriver(driverName, driverPath, rawDriver)
	if err != nil {
		return nil, err
	}

	privKey, pubKey, err := ssh.GenerateKeys()
	if err != nil {
		return nil, err
	}

	return &host.Host{
		ConfigVersion: host.Version,
		Name:          driver.GetMachineName(),
		Driver:        driver,
		DriverName:    driver.DriverName(),
		DriverPath:    driverPath,
		RawDriver:     rawDriver,
		PublicKey:     pubKey,
		PrivateKey:    privKey,
	}, nil
}

func (api *Client) Load(name string) (*host.Host, error) {
	h, err := api.Filestore.Load(name)
	if err != nil {
		return nil, err
	}

	d, err := api.clientDriverFactory.NewRPCClientDriver(h.DriverName, h.DriverPath, h.RawDriver)
	if err != nil {
		return nil, err
	}
	h.Driver = d
	return h, nil
}
