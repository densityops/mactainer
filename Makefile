
# brew install opam libev libffi pkg-config autoconf automake libtool dylibbundler wget

BUILD_DIR := _build
BUILD_DIR_BIN := $(BUILD_DIR)/bin
DEPS_DIR := $(abspath deps)

LINUXKIT := $(abspath $(BUILD_DIR_BIN)/linuxkit)
HYPERKIT := $(abspath $(BUILD_DIR_BIN)/hyperkit)
VPNKIT := $(abspath $(BUILD_DIR_BIN)/vpnkit)

.PHONY: update-submodules build-deps

clean:
	

build-deps: $(LINUXKIT) $(HYPERKIT) $(VPNKIT) update-submodules

.PHONY: build
build: build-deps build-dir
	@echo "build done"

.PHONY: build-dir
build-dir:
	@mkdir -p _build/bin

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
		cp bin/linuxkit ../../_build/bin/linuxkit

$(HYPERKIT):
	@cd $(DEPS_DIR)/hyperkit && \
		$(MAKE) all && \
		cp build/hyperkit ../../_build/bin/hyperkit
	
$(VPNKIT):
	@cd $(DEPS_DIR)/vpnkit && \
		export OPAMVERBOSE=1 && \
		export OPAMYES=1 && \
		$(MAKE) -f Makefile.darwin ocaml && \
		$(MAKE) -f Makefile.darwin depends && \
		$(MAKE) -f Makefile.darwin build && \
		cp _build/install/default/bin/vpnkit ../../_build/bin/vpnkit