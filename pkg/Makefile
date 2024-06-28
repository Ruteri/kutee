.PHONY: all
all:
	exit 1

.PHONY: ratls-image
ratls-image:
	docker build -t ratls:latest -f pkg/ratls.dockerfile .

.PHONY: simple-key-service-image
simple-key-service-image:
	cd pkg/simple-key-service && $(MAKE) docker-httpserver

.PHONY: simple-key-service-image
simple-key-service-image:
	cd pkg/simple-key-service && $(MAKE) docker-httpserver
