
# brew install opam libev libffi pkg-config autoconf automake libtool dylibbundler wget docker linux-headers
# ./System/Volumes/Data/Library/Application\ Support/com.canonical.multipass/bin/qemu-img convert -f qcow2 -O qcow2 -o lazy_refcounts=on,compat=1.1,preallocation=metadata -p ../_build/instances/fedora-coreos-34.20210919.3.0-qemu.x86_64.qcow2 ../_build/instances/mactainer/mactainer.qcow2
# /System/Volumes/Data/Library/Application\ Support/com.canonical.multipass/bin/qemu-img convert -f qcow2 -O qcow2 -o lazy_refcounts=on,compat=1.1,preallocation=metadata -p ../_build/instances/mactainer/fedora-coreos-34.20210919.3.0-qemu.x86_64.qcow2 ../_build/instances/mactainer/mactainer.qcow2
# Add portainer
# qemu-system-x86_64 -m 2048 -accel hvf -nographic \
 -drive if=virtio,file=_build/instances/fedora-coreos-34.20210919.3.0-qemu.x86_64.qcow2 \
 -nic user,model=virtio,hostfwd=tcp::2223-:22 \
 -fw_cfg name=opt/com.coreos/config,file=_build/instances/mactainer/mactainer.ign
# butane -o ../../../_build/instances/mactainer/mactainer.ign ../../../assets/ignition/mct.bu 
# _build/bin/gvproxy --listen unix:///Users/mabe01/code/src/github.com/densityops/mactainer/_build/run/api.sock --listen-vpnkit unix:///Users/mabe01/code/src/github.com/densityops/mactainer/_build/run/tap.vsock
# cd ../mct-bundler/ && go build -ldflags="-X 'github.com/densityops/mactainer/mct/pkg/bundle.Version=v1.0.0'" . && cd - && go build . && ./mct setup
# ssh -l core localhost -p 2022 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null
#  TODO
# fork machine from crc, chanfe binary for hyperkit driver
# fork hyperkit driver from crc, add guest forwards, add 9p fs.
# add vsudd to ignition
# Recreate kernel and initrd copy with qemu (scp)
# /System/Volumes/Data/Library/Application\ Support/com.canonical.multipass/bin/qemu-img convert -f qcow2 -O qcow2 -o lazy_refcounts=on -p ../../../_build/instances/fedora-coreos-34.20210919.3.0-qemu.x86_64.qcow2 ../../pkg/bundle/files/machine/image.qcow2
# TZ=UTC git --no-pager show \
  --quiet \
  --abbrev=12 \
  --date='format-local:%Y%m%d%H%M%S' \
  --format="%cd-%h"

OPAMROOT ?= ~/.opam
OPAM_COMP := 4.11.1
BUNDLE_VERSION := v1.0.0
MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
ROOT_DIR := $(dir $(MAKEFILE_PATH))
BUILD_DIR := $(abspath mct/pkg/bundle/files)
BUILD_DIR_BIN := $(BUILD_DIR)/bin
DEPS_DIR := $(abspath deps)
IMAGE_DIR := $(abspath $(ROOT_DIR)/assets/images)

HYPERKIT := $(abspath $(BUILD_DIR_BIN)/hyperkit)
HYPERKIT_DRIVER := $(abspath $(BUILD_DIR_BIN)/machine-driver-hyperkit)
QCOW_TOOL := $(abspath $(BUILD_DIR_BIN)/qcow-tool)




# in container: socat - VSOCK-LISTEN:1525
# on host: echo "hello"|socat - UNIX:_build/run/mactainer/guest.000005f5 

# in container:

.PHONY: bundle
bundle:
	@cd $(ROOT_DIR)/mct/cmd/mct-bundler && \
		 go build -ldflags="-X 'github.com/densityops/mactainer/mct/pkg/bundle.Version=$(BUNDLE_VERSION)'" -o $(BUILD_DIR_BIN)/mct-bundle-$(BUNDLE_VERSION) .


.PHONY: mct
mct:
	@cd $(ROOT_DIR)/mct/cmd/mct && \
		go build -o $(BUILD_DIR_BIN)/mct .

.PHONY: build
build: mct bundle build-deps build-dir
	@echo "build done"

.PHONY: build-dir
build-dir:
	@mkdir -p $(BUILD_DIR)/bin
	@mkdir -p $(BUILD_DIR)/files


.PHONY: install
install: build
	install -d $${HOME}/.mct/bundles
	@install $(BUILD_DIR_BIN)/mct-bundle-$(BUNDLE_VERSION) $${HOME}/.mct/bundles/mct-bundle-$(BUNDLE_VERSION)
	@install $(BUILD_DIR_BIN)/mct /usr/local/bin/mct

.PHONY: run-docker
run-docker:
	@docker -H unix://$(BUILD_DIR)/run/mactainer/guest.00000948 run -p 1345:1245 --net host --privileged --rm  -it alpine /bin/ash

.PHONY: build-and-push-images
build-and-push-images:
	@cd $(DEPS_DIR)/vpnkit && \
		docker build -t quay.io/densityops/vpnkit-expose-port:$(VPNKIT_GIT_VERSION) -f $(IMAGE_DIR)/Dockerfile.vpnkit-expose-port . && \
		docker push quay.io/densityops/vpnkit-expose-port:$(VPNKIT_GIT_VERSION) && \
		docker build -t quay.io/densityops/vpnkit-forwarder:$(VPNKIT_GIT_VERSION) -f $(IMAGE_DIR)/Dockerfile.vpnkit-forwarder . && \
		docker push quay.io/densityops/vpnkit-forwarder:$(VPNKIT_GIT_VERSION)
		
	@cd $(DEPS_DIR)/vpnkit/c/vpnkit-tap-vsockd && \
		docker build -t quay.io/densityops/vpnkit-tap-vsockd:$(VPNKIT_GIT_VERSION) -f $(IMAGE_DIR)/Dockerfile.vpnkit-tap-vsockd . && \
		docker push quay.io/densityops/vpnkit-tap-vsockd:$(VPNKIT_GIT_VERSION)

	@cd $(DEPS_DIR)/vpnkit/c/vpnkit-9pmount-vsock && \
		docker build -t quay.io/densityops/vpnkit-9pmount-vsock:$(VPNKIT_GIT_VERSION) -f $(IMAGE_DIR)/Dockerfile.vpnkit-9pmount-vsock . && \
		docker push quay.io/densityops/vpnkit-9pmount-vsock:$(VPNKIT_GIT_VERSION)

	@cd $(DEPS_DIR)/virtsock && \
		docker build -t quay.io/densityops/vsudd:$(VIRTSOCK_GIT_VERSION) -f $(IMAGE_DIR)/Dockerfile.vsudd . && \
		docker push quay.io/densityops/vsudd:$(VIRTSOCK_GIT_VERSION)

.PHONY: clean
clean: clean-deps
	@rm -fr $(BUILD_DIR)/bin

.PHONY: clean-deps
clean-deps:
	@cd $(DEPS_DIR) && \
		for dir in $$(ls -1); do cd $${dir} && make clean; cd .. ; done

.PHONY: build-deps
build-deps: $(QCOW_TOOL) $(HYPERKIT) $(HYPERKIT_DRIVER)

.PHONY: update-submodules
update-submodules:
	@if git submodule status | egrep -q '^[-]|^[+]' ; then \
		echo "INFO: Need to reinitialize git submodules"; \
		git submodule update --init; \
	fi

.PHONY: hyperkit
hyperkit: $(HYPERKIT)
	$(HYPERKIT)


.PHONY: hyperkit_driver
hyperkit_driver: $(HYPERKIT_DRIVER)
	$(HYPERKIT_DRIVER)

.PHONY: qcow_tool
qcow_tool: $(QCOW_TOOL)
	$(QCOW_TOOL)

$(HYPERKIT):
	@cd $(DEPS_DIR)/hyperkit && \
		export OPAMVERBOSE=1 && \
		export OPAMYES=1 && \
		export OPAM_COMP=$(OPAM_COMP) && \
		export MACOSX_DEPLOYMENT_TARGET="10.11" && \
		opam init -v -n --comp="$${OPAM_COMP}" --switch="$${OPAM_COMP}" && \
		opam pin add qcow.0.11.0 git://github.com/mirage/ocaml-qcow -n && \
		opam pin add qcow-tool.0.11.0 git://github.com/mirage/ocaml-qcow -n && \
		opam pin add hyperkit . && \
		opam config exec -- make clean && \
        opam config exec -- make all && \
		cp build/hyperkit $(BUILD_DIR_BIN)/hyperkit
	

$(HYPERKIT_DRIVER):
	@cd $(DEPS_DIR)/machine-driver-hyperkit && \
		go mod tidy && \
		go mod vendor && \
		GOOS=darwin go build \
		-installsuffix "static" \
		-ldflags="-s -w" \
		-o $(BUILD_DIR_BIN)/machine-driver-hyperkit && \
	    chmod +x $(BUILD_DIR_BIN)/machine-driver-hyperkit

$(QCOW_TOOL):
	@cp $(OPAMROOT)/$(OPAM_COMP)/bin/qcow-tool $(BUILD_DIR_BIN)/qcow-tool