MINIKUBE_SHA256=386eb267e0b1c1f000f1b7924031557402fffc470432dc23b9081fc6962fd69b
.PHONY: minikube-setup
minikube-setup:
	curl -LO https://github.com/kubernetes/minikube/releases/download/v1.33.1/minikube-linux-amd64
	SHA256_ACTUAL=`sha256sum minikube-linux-amd64`; \
        case "$$SHA256_ACTUAL" in \
            ($(MINIKUBE_SHA256)\ *) : ok ;; \
            (*) echo minikube checksum mismatch, expected=\"$(MINIKUBE_SHA256)\" actual=\"$$SHA256_ACTUAL\"; \
            exit 1 ;; \
        esac
	sudo install minikube-linux-amd64 /usr/local/bin/minikube && rm minikube-linux-amd64
	minikube start --container-runtime=containerd --docker-opt containerd=/var/run/containerd/containerd.sock
	minikube addons enable gvisor


.PHONY: minikube-delegate-cpu
minikube-delegate-cpu:
	sudo mkdir -p /etc/systemd/system/user@.service.d
	echo "[Service]" | sudo tee /etc/systemd/system/user@.service.d/delegate.conf
	echo "Delegate=cpu cpuset io memory pids" | sudo tee -a /etc/systemd/system/user@.service.d/delegate.conf
	sudo systemctl daemon-reload
