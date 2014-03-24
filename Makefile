.PHONY: default server client godeps fmt clean all release-all release-server release-client contributors tor openssl
export GOPATH:=$(shell pwd)

VENDOR_PATH=$(GOPATH)/src/vendor

TOR_PATH=$(VENDOR_PATH)/tor-0.2.4.21
OPENSSL_PATH=$(VENDOR_PATH)/openssl-1.0.1f
ZLIB_PATH=$(VENDOR_PATH)/zlib-1.2.8
LIBEVENT_PATH=$(VENDOR_PATH)/libevent-2.0.21-stable

TOR_MAKEFILE=$(TOR_PATH)/Makefile
OPENSSL_MAKEFILE=$(OPENSSL_PATH)/Makefile
ZLIB_MAKEFILE=$(ZLIB_PATH)/Makefile
LIBEVENT_MAKEFILE=$(LIBEVENT_PATH)/Makefile

LIBTOR=$(TOR_PATH)/src/or/libtor.a
LIBCURVE=$(TOR_PATH)/src/common/libcurve25519_donna.a
LIBOR_CRYPTO=$(TOR_PATH)/src/common/libor-crypto.a
LIBOR_EVENT=$(TOR_PATH)/src/common/libor-event.a
LIBOR=$(TOR_PATH)/src/common/libor.a
LIBZ=$(ZLIB_PATH)/libz.a
LIBSSL=$(OPENSSL_PATH)/libssl.a
LIBCRYPTO=$(OPENSSL_PATH)/libcrypto.a
LIBEVENT=$(LIBEVENT_PATH)/build/lib/libevent.a
ALL_LIBS=$(LIBZ) $(LIBSSL) $(LIBCRYPTO) $(LIBEVENT) $(LIBTOR) $(LIBCURVE) $(LIBOR_CRYPTO) $(LIBOR_EVENT) $(LIBOR)

TOR?=0
RELEASE_TAG?=debug
TOR_TAG=tor

ifeq "$(TOR)" "0"
  ALL_LIBS=
  TOR_TAG=
endif

# poor man's platform-specific rules, this obviously breaks windows
OS=$(shell uname -s)
OPENSSL_CONFIGURE=./config
CGO_LDFLAGS=-Wl,--start-group $(ALL_LIBS) -Wl,--end-group -Wl,-Bstatic -lm -lrt -Wl,-Bdynamic -lpthread -lc
ifeq "$(OS)" "Darwin"
	CGO_LDFLAGS=$(ALL_LIBS)
	OPENSSL_CONFIGURE=./Configure darwin64-x86_64-cc
endif


BUILDTAGS=$(RELEASE_TAG) $(TOR_TAG)

default: all

$(ZLIB_MAKEFILE):
	cd $(ZLIB_PATH) && CFLAGS="-fPIC" ./configure
	
$(LIBZ): $(ZLIB_MAKEFILE)
	$(MAKE) -C $(ZLIB_PATH)

$(LIBEVENT_MAKEFILE):
	cd $(LIBEVENT_PATH) && ./configure --disable-shared --enable-static --with-pic --prefix=$(LIBEVENT_PATH)/build

$(LIBEVENT): $(LIBEVENT_MAKEFILE)
	$(MAKE) -C $(LIBEVENT_PATH) install

$(LIBSSL): openssl
$(LIBCRYPTO): openssl

$(OPENSSL_MAKEFILE):
	cd $(OPENSSL_PATH) && $(OPENSSL_CONFIGURE) no-shared no-dso -fPIC

openssl: $(OPENSSL_MAKEFILE)
	$(MAKE) -C $(OPENSSL_PATH)

$(LIBOR): tor
$(LIBOR_EVENT): tor
$(LIBOR_CRYPTO): tor
$(LIBTOR): tor
$(LIBCURVE): tor

$(TOR_MAKEFILE):
	cd $(TOR_PATH) && CFLAGS="-fPIC" ./configure --enable-static-libevent --enable-static-zlib --with-libevent-dir=$(LIBEVENT_PATH)/build --with-zlib-dir=$(ZLIB_PATH) --enable-static-openssl --with-openssl-dir=$(OPENSSL_PATH)

tor: $(TOR_MAKEFILE)
	$(MAKE) -C $(TOR_PATH)

godeps:
	go get -tags '$(BUILDTAGS)' -d -v shroud/...

fmt:
	go fmt shroud/...


discover: godeps
	go install -gcflags "-N -l" -tags '$(BUILDTAGS)' shroud/cmd/shroud-discover

server: godeps
	go install -gcflags "-N -l" -tags '$(BUILDTAGS)' shroud/cmd/shroud-server

# XXX
# normally you can just put this in a #cgo pragma, but you can't use relative paths to the libraries
# until go 1.3 in that way. so instead we'll just pass the in via CGO_LDFLAGS
# where at least we can resolve $GOPATH to get us absolute paths. it's a little hacky
client: godeps $(ALL_LIBS)
	CGO_LDFLAGS="$(CGO_LDFLAGS)" go install -gcflags "-N -l" -tags '$(BUILDTAGS)' shroud/cmd/shroud

release-client: RELEASE_TAG=release
release-client: client

release-server: RELEASE_TAG=release
release-server: server

release-server: RELEASE_TAG=release
release-discover: discover

release-all: fmt release-client release-server release-discover

all: fmt client server discover

goclean:
	go clean -i -r shroud/...

clean: goclean
	$(MAKE) -C $(ZLIB_PATH) clean || true
	$(MAKE) -C $(OPENSSL_PATH) clean || true
	$(MAKE) -C $(LIBEVENT_PATH) clean || true
	$(MAKE) -C $(TOR_PATH) clean || true

contributors:
	echo "Contributors to shroud:\n" > CONTRIBUTORS
	git log --raw | grep "^Author: " | sort | uniq | cut -d ' ' -f2- | sed 's/^/- /' | cut -d '<' -f1 >> CONTRIBUTORS
