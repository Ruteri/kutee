#!/bin/bash
set -e

dpkg -i /tmp/kutee/containerd.io-1.6.33.deb /tmp/kutee/docker-ce-26.deb /tmp/kutee/docker-ce-cli-26.deb

mkdir -p /etc/systemd/system/user@.service.d
cat >/etc/systemd/system/user@.service.d/delegate.conf <<EOF
[Service]
Delegate=cpu cpuset io memory pids
EOF
systemctl daemon-reload

cat >/usr/local/bin/kutee-start <<EOF 
#!/bin/bash
set -e
minikube start --container-runtime=containerd --docker-opt containerd=/var/run/containerd/containerd.sock
minikube image load /kutee/simple-key-service.tar
minikube image load /kutee/ratls.tar
minikube addons enable gvisor
docker image load -i /kutee/ratls.tar
docker run --net host --rm -d ratls /app/ratls -server -target-domain http://localhost -target-port 8087 -listen-port 8080
cd /home/tdx && kutee-orchestrator --listen-addr 0.0.0.0:8087
EOF
chmod +x /usr/local/bin/kutee-start

cat >/etc/systemd/system/kutee.service <<EOF
[Unit]
Description=Kutee simple service
Wants=network.target
After=syslog.target network-online.target
[Service]
Type=simple
User=tdx
ExecStart=/usr/local/bin/kutee-start
Restart=never
KillMode=process
[Install]
WantedBy=multi-user.target
EOF

chmod 640 /etc/systemd/system/kutee.service
systemctl daemon-reload
systemctl enable kutee.service

# Configure docker
sudo usermod -aG docker tdx

# Cleanup
rm -rf /tmp/kutee || true
