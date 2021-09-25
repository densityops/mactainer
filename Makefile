
# brew install opam libev libffi pkg-config autoconf automake libtool dylibbundler wget

BUILD_DIR := _build
BUILD_DIR_BIN := $(BUILD_DIR)/bin
DEPS_DIR := $(abspath deps)

LINUXKIT := $(abspath $(BUILD_DIR_BIN)/linuxkit)
HYPERKIT := $(abspath $(BUILD_DIR_BIN)/hyperkit)
VPNKIT := $(abspath $(BUILD_DIR_BIN)/vpnkit)

.PHONY: build-instances
build-instances: build-dir
	@$(LINUXKIT) build -dir $(BUILD_DIR)/instances/mactainer -format kernel+initrd instances/mactainer.yml
	@cp assets/uefi/UEFI.fd $(BUILD_DIR)/instances/mactainer
	@cp assets/metadata/metadata.json $(BUILD_DIR)/instances/mactainer


.PHONY: run-instances
run-instances:
	@mkdir -p $(BUILD_DIR)/run/mactainer
	@$(LINUXKIT) run hyperkit \
		-networking=vpnkit \
		-vsock-ports=2376 \
		-disk size=4096M \
		-data-file $(BUILD_DIR)/instances/mactainer/metadata.json \
		-vpnkit=$(VPNKIT) \
		-cpus 2 \
		-mem 2048  \
		-uefi \
		-fw $(BUILD_DIR)/instances/mactainer/UEFI.fd \
		-state $(BUILD_DIR)/run/mactainer \
		$(BUILD_DIR)/instances/mactainer/mactainer
	@rm -fr $(BUILD_DIR)/run/mactainer

.PHONY: build
build: build-deps build-dir
	@echo "build done"

.PHONY: build-dir
build-dir:
	@mkdir -p $(BUILD_DIR)/bin
	@mkdir -p $(BUILD_DIR)/instances/mactainer
	@mkdir -p $(BUILD_DIR)/run

.PHONY: clean
clean: clean-deps
	@rm -fr _build

.PHONY: clean-deps
clean-deps:
	@cd $(DEPS_DIR) && \
		for dir in $$(ls -1); do cd $${dir} && make clean; cd .. ; done

.PHONY: build-deps
build-deps: $(LINUXKIT) $(HYPERKIT) $(VPNKIT)

.PHONY: update-submodules
update-submodules:
	@if git submodule status | egrep -q '^[-]|^[+]' ; then \
		echo "INFO: Need to reinitialize git submodules"; \
		git submodule update --init; \
	fi

.PHONY: linuxkit
linuxkit: $(LINUXKiT)
	$(LINUXKIT)

.PHONY: hyperkit
hyperkit: $(HYPERKIT)
	$(HYPERKIT)

.PHONY: vpnkit
vpnkit: $(VPNKIT)
	$(VPNKIT)

$(LINUXKIT):
	@cd $(DEPS_DIR)/linuxkit && \
		$(MAKE) all && \
		cp bin/linuxkit ../../$(BUILD_DIR)/bin/linuxkit

$(HYPERKIT):
	@cd $(DEPS_DIR)/hyperkit && \
		$(MAKE) all && \
		cp build/hyperkit ../../$(BUILD_DIR)/bin/hyperkit
	
$(VPNKIT):
	@cd $(DEPS_DIR)/vpnkit && \
		export OPAMVERBOSE=1 && \
		export OPAMYES=1 && \
		$(MAKE) -f Makefile.darwin ocaml && \
		$(MAKE) -f Makefile.darwin depends && \
		$(MAKE) -f Makefile.darwin build && \
		cp $(BUILD_DIR)/install/default/bin/vpnkit ../../$(BUILD_DIR)/bin/vpnkit