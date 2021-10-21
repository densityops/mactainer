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
package persist

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/densityops/mactainer/mct/pkg/drivers/none"
	"github.com/densityops/mactainer/mct/pkg/libmachine/host"
	"github.com/stretchr/testify/assert"
)

func getTestStore() (Filestore, func(), error) {
	tmpDir, err := ioutil.TempDir("", "machine-test-")
	if err != nil {
		return Filestore{}, nil, err
	}
	return Filestore{
			MachinesDir: tmpDir,
		}, func() {
			os.RemoveAll(tmpDir)
		}, nil
}

func TestStoreSave(t *testing.T) {
	store, cleanup, err := getTestStore()
	assert.NoError(t, err)
	defer cleanup()

	h := testHost()

	assert.NoError(t, store.Save(h))

	path := filepath.Join(store.MachinesDir, h.Name)
	assert.DirExists(t, path)
}

func TestStoreSaveOmitRawDriver(t *testing.T) {
	store, cleanup, err := getTestStore()
	assert.NoError(t, err)
	defer cleanup()

	h := testHost()

	assert.NoError(t, store.Save(h))

	configJSONPath := filepath.Join(store.MachinesDir, h.Name, "config.json")

	configData, err := ioutil.ReadFile(configJSONPath)
	assert.NoError(t, err)

	fakeHost := make(map[string]interface{})

	assert.NoError(t, json.Unmarshal(configData, &fakeHost))

	_, ok := fakeHost["RawDriver"]
	assert.False(t, ok)
}

func TestStoreRemove(t *testing.T) {
	store, cleanup, err := getTestStore()
	assert.NoError(t, err)
	defer cleanup()

	h := testHost()

	assert.NoError(t, store.Save(h))

	path := filepath.Join(store.MachinesDir, h.Name)
	assert.DirExists(t, path)

	err = store.Remove(h.Name)
	assert.NoError(t, err)

	assert.NoDirExists(t, path)
}

func TestStoreExists(t *testing.T) {
	store, cleanup, err := getTestStore()
	assert.NoError(t, err)
	defer cleanup()

	h := testHost()

	exists, err := store.Exists(h.Name)
	assert.NoError(t, err)
	assert.False(t, exists)

	assert.NoError(t, store.Save(h))

	assert.NoError(t, store.SetExists(h.Name))

	exists, err = store.Exists(h.Name)
	assert.NoError(t, err)
	assert.True(t, exists)

	assert.NoError(t, store.Remove(h.Name))

	exists, err = store.Exists(h.Name)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestStoreLoad(t *testing.T) {
	store, cleanup, err := getTestStore()
	assert.NoError(t, err)
	defer cleanup()

	h := testHost()

	assert.NoError(t, store.Save(h))

	h, err = store.Load(h.Name)
	assert.NoError(t, err)

	rawDataDriver, ok := h.Driver.(*host.RawDataDriver)
	assert.True(t, ok)

	realDriver := none.NewDriver(h.Name, store.MachinesDir)
	assert.NoError(t, json.Unmarshal(rawDataDriver.Data, &realDriver))
}

func testHost() *host.Host {
	return &host.Host{
		ConfigVersion: host.Version,
		Name:          "test-host",
		Driver:        none.NewDriver("test-host", "/tmp/artifacts"),
		DriverName:    "none",
	}
}
