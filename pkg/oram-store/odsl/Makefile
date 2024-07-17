CC?=/usr/bin/gcc
CXX?=/usr/bin/g++
INSTALL?=sudo install

LD_PATH?=/usr/local/lib/

all: go/libodsl.so install

.PHONY: go/libodsl.so
go/libodsl.so:
	CC=$(CC) CXX=$(CXX) cmake -B build -G Ninja -DCMAKE_BUILD_TYPE=Release
	CC=$(CC) CXX=$(CXX) ninja -C build
	cp build/odsl.go odsl.go

install: go/libodsl.so
	$(INSTALL) go/libodsl.so $(LD_PATH)

clean:
	rm -rf ./go
	rm -rf ./build
