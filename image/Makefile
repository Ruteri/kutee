VERSION?=0.0.1
DOCKER?=docker
VIRT_CUSTOMIZE?=virt-customize

minikube:
	# Pulls to cwd rather than to ../minikube
	$(MAKE) -f ../minikube/Makefile minikube-linux-amd64
	mv minikube-linux-amd64 minikube
	chmod +x minikube

.PHONY: kutee-orchestrator
kutee-orchestrator:
	cd ../pkg/kutee-orchestrator && go build -trimpath -ldflags "-X github.com/flashbots/go-template/common.Version=${VERSION}" -v -o ../../image/kutee-orchestrator cmd/httpserver/main.go

simple-key-service.tar:
	cd ../pkg/simple-key-service && $(MAKE) docker-httpserver
	$(DOCKER) image save simple-key-service -o simple-key-service.tar

ratls.tar:
	docker build -t ratls -f ../pkg/ratls.dockerfile .
	$(DOCKER) image save ratls -o ratls.tar

auth_users.json:
	@echo '{"test": "dGVzdOOwxEKY/BwUmvv0yJlvuSQnrkHkZJuTTKSVmRt4UrhV"}' >auth_users.json

workload.yaml:
	@touch workload.yaml

docker-ce-26.deb:
	@curl https://download.docker.com/linux/ubuntu/dists/noble/pool/stable/amd64/docker-ce_26.1.4-1~ubuntu.24.04~noble_amd64.deb -o docker-ce-26.deb

docker-ce-cli-26.deb:
	@curl https://download.docker.com/linux/ubuntu/dists/noble/pool/stable/amd64/docker-ce-cli_26.1.4-1~ubuntu.24.04~noble_amd64.deb -o docker-ce-cli-26.deb

containerd.io-1.6.33.deb:
	@curl https://download.docker.com/linux/ubuntu/dists/noble/pool/stable/amd64/containerd.io_1.6.33-1_amd64.deb -o containerd.io-1.6.33.deb

PWD=$(shell pwd)
.PHONY: kutee-image
kutee-image: minikube kutee-orchestrator auth_users.json simple-key-service.tar ratls.tar workload.yaml docker-ce-26.deb containerd.io-1.6.33.deb docker-ce-cli-26.deb
	install -m 0777 ${IMAGE_PATH} /tmp/$(shell basename $(IMAGE_PATH))
	$(VIRT_CUSTOMIZE) -a /tmp/$(shell basename $(IMAGE_PATH)) \
        --mkdir /kutee/ \
        --mkdir /tmp/kutee/ \
        --mkdir /usr/local/bin/ \
        --copy-in containerd.io-1.6.33.deb:/tmp/kutee/ \
        --copy-in docker-ce-26.deb:/tmp/kutee/ \
        --copy-in docker-ce-cli-26.deb:/tmp/kutee/ \
        --copy-in minikube:/usr/local/bin/ \
        --copy-in kutee-orchestrator:/usr/local/bin/ \
        --copy-in setup.sh:/tmp/kutee/ \
        --run-command "/tmp/kutee/setup.sh"
	mv /tmp/$(shell basename $(IMAGE_PATH)) .
