#remove leading characters since package version expects to start with digit
PACKAGE_VERSION ?= $(shell echo $(VERSION) | sed 's/^[a-z]*-//' | sed 's/-//')

# Creates the proxy debian packages. BUILD_WITH_CONTAINER=1 or in CI/CD (BUILD_WITH_CONTAINER=0)
deb: ${TARGET_OUT_LINUX}/release/istio-sidecar.deb

# fpm likes to add extremely high levels of compression. This is fine for release, but for local runs
# where we are just pushing to a local registry (compressed again!), it adds ~1min to builds.
ifneq ($(FAST_VM_BUILDS),)
DEB_COMPRESSION=--deb-compression=none
RPM_COMPRESSION=--rpm-compression=none
endif

# Base directory for istio binaries. Likely to change !
ISTIO_DEB_BIN=/usr/local/bin

# Home directory of istio-proxy user. It is symlinked /etc/istio --> /var/lib/istio
ISTIO_PROXY_HOME=/var/lib/istio

ISTIO_DEB_DEPS:=pilot-discovery istioctl
ISTIO_FILES:=
$(foreach DEP,$(ISTIO_DEB_DEPS),\
        $(eval ${TARGET_OUT_LINUX}/release/istio.deb: $(TARGET_OUT_LINUX)/$(DEP)) \
        $(eval ISTIO_FILES+=$(TARGET_OUT_LINUX)/$(DEP)=$(ISTIO_DEB_BIN)/$(DEP)) )

SIDECAR_DEB_DEPS:=envoy pilot-agent
SIDECAR_FILES:=
$(foreach DEP,$(SIDECAR_DEB_DEPS),\
        $(eval ${TARGET_OUT_LINUX}/release/istio-sidecar.deb: $(TARGET_OUT_LINUX)/$(DEP)) \
        $(eval ${TARGET_OUT_LINUX}/release/istio-sidecar.rpm: $(TARGET_OUT_LINUX)/$(DEP)) \
        $(eval ${TARGET_OUT_LINUX}/release/istio-sidecar-rhel.rpm: $(TARGET_OUT_LINUX)/$(DEP)) \
        $(eval SIDECAR_FILES+=$(TARGET_OUT_LINUX)/$(DEP)=$(ISTIO_DEB_BIN)/$(DEP)) )

ISTIO_DEB_DEST:=${ISTIO_DEB_BIN}/istio-start.sh \
		/lib/systemd/system/istio.service \
		/var/lib/istio/envoy/sidecar.env

$(foreach DEST,$(ISTIO_DEB_DEST),\
        $(eval ${TARGET_OUT_LINUX}/istio-sidecar.deb:   tools/packaging/common/$(notdir $(DEST))) \
        $(eval SIDECAR_FILES+=${REPO_ROOT}/tools/packaging/common/$(notdir $(DEST))=$(DEST)))

SIDECAR_FILES+=${REPO_ROOT}/tools/packaging/common/envoy_bootstrap.json=/var/lib/istio/envoy/envoy_bootstrap_tmpl.json

# original name used in 0.2 - will be updated to 'istio.deb' since it now includes all istio binaries.
SIDECAR_PACKAGE_NAME ?= istio-sidecar

# TODO: rename istio-sidecar.deb to istio.deb

# Note: adding --deb-systemd ${REPO_ROOT}/tools/packaging/common/istio.service will result in
# a /etc/systemd/system/multi-user.target.wants/istio.service and auto-start. Currently not used
# since we need configuration.
# --iteration 1 adds a "-1" suffix to the version that didn't exist before
${TARGET_OUT_LINUX}/release/istio-sidecar.deb: | ${TARGET_OUT_LINUX} deb/fpm
${TARGET_OUT_LINUX}/release/istio-sidecar.rpm: | ${TARGET_OUT_LINUX} rpm/fpm
${TARGET_OUT_LINUX}/release/istio-sidecar-rhel.rpm: | ${TARGET_OUT_LINUX} rhel-rpm/fpm

# Package the sidecar rpm file.
rpm/fpm: ambient_rpm rhel-rpm/fpm
	rm -f ${TARGET_OUT_LINUX}/release/istio-sidecar.rpm
	fpm -s dir -t rpm -n ${SIDECAR_PACKAGE_NAME} -p ${TARGET_OUT_LINUX}/release/istio-sidecar.rpm --version $(PACKAGE_VERSION) -f \
		--url https://solo.io  \
		--license Apache \
		--architecture "${TARGET_ARCH}" \
		--vendor solo.io \
		--maintainer support@solo.io \
		--after-install tools/packaging/postinst.sh \
		--config-files /var/lib/istio/envoy/envoy_bootstrap_tmpl.json \
		--config-files /var/lib/istio/envoy/sidecar.env \
		--description "Istio Sidecar" \
		--depends iproute \
		--depends iptables \
		--depends sudo \
		--depends hostname \
		$(RPM_COMPRESSION) \
		$(SIDECAR_FILES)

# We need to define the digest algorithm for RHEL+FIPS, the rpm command reports SHA256 digest OK outside of FIPS
# and the default seems to be MD5, which is an algorithm not allowed in FIPS mode. I also don't want to change the
# main rpm packaging because I don't know what may break if that is changed to use the SHA256 digest. Things indicate
# that it might not be an issue, but that's a risk.
rhel-rpm/fpm:
	rm -f ${TARGET_OUT_LINUX}/release/istio-sidecar-rhel.rpm
	fpm -s dir -t rpm -n ${SIDECAR_PACKAGE_NAME} -p ${TARGET_OUT_LINUX}/release/istio-sidecar-rhel.rpm --version $(PACKAGE_VERSION) \
		--rpm-digest sha256 \
		-f \
		--url https://solo.io  \
		--license Apache \
		--architecture "${TARGET_ARCH}" \
		--vendor solo.io \
		--maintainer support@solo.io \
		--after-install tools/packaging/postinst.sh \
		--config-files /var/lib/istio/envoy/envoy_bootstrap_tmpl.json \
		--config-files /var/lib/istio/envoy/sidecar.env \
		--description "Istio Sidecar" \
		--depends iproute \
		--depends iptables \
		--depends sudo \
		--depends hostname \
		$(RPM_COMPRESSION) \
		$(SIDECAR_FILES)
ifeq ($(CI),true)
	mkdir -p ${GOPATH}/../out/rpm
ifeq ($(TARGET_ARCH),amd64)
	cp ${TARGET_OUT_LINUX}/release/istio-sidecar-rhel.rpm ${GOPATH}/../out/rpm
else
	cp ${TARGET_OUT_LINUX}/release/istio-sidecar-rhel.rpm ${GOPATH}/../out/rpm/istio-sidecar-rhel-$(TARGET_ARCH).rpm
endif
endif

# Package the sidecar deb file.
deb/fpm: ambient_deb
	rm -f ${TARGET_OUT_LINUX}/release/istio-sidecar.deb
	fpm -s dir -t deb -n ${SIDECAR_PACKAGE_NAME} -p ${TARGET_OUT_LINUX}/release/istio-sidecar.deb --version $(PACKAGE_VERSION) -f \
		--url https://solo.io  \
		--license Apache \
		--vendor solo.io \
		--architecture "${TARGET_ARCH}" \
		--maintainer support@solo.io \
		--after-install tools/packaging/postinst.sh \
		--config-files /var/lib/istio/envoy/envoy_bootstrap_tmpl.json \
		--config-files /var/lib/istio/envoy/sidecar.env \
		--description "Istio Sidecar" \
		--depends iproute2 \
		--depends iptables \
		--depends sudo \
		--depends hostname \
		$(DEB_COMPRESSION) \
		$(SIDECAR_FILES)

.PHONY: \
	deb \
	deb/fpm \
	rpm/fpm \
	rhel-rpm/fpm \
	sidecar.deb
