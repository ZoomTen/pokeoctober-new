# This is now just a front to run the Ninja build system.

GO ?= go

.PHONY: all clean tidy

all:
	cd tools && ninja
	$(GO) run utils/configure.go
	cd build && ninja

clean:
	rm -rf build

tidy:
	rm -rf build
	cd tools && ninja -t clean