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
	"io"

	rpcdriver "github.com/densityops/machine/libmachine/drivers/rpc"
	"github.com/densityops/mactainer/mct/pkg/libmachine/host"
	"github.com/densityops/mactainer/mct/pkg/libmachine/persist"
)

type API interface {
	io.Closer
	NewHost(driverName string, driverPath string, rawDriver []byte) (*host.Host, error)
	persist.Store
}

type Client struct {
	*persist.Filestore
	clientDriverFactory rpcdriver.RPCClientDriverFactory
}

func NewClient(storePath string) *Client {
	return &Client{
		Filestore:           persist.NewFilestore(storePath),
		clientDriverFactory: rpcdriver.NewRPCClientDriverFactory(),
	}
}

func (api *Client) Close() error {
	return api.clientDriverFactory.Close()
}
