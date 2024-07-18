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

find /kutee/ -name "*.tar" -exec minikube image load {} \;

minikube addons enable gvisor

cp /kutee/deployment.yaml /home/tdx/workload.yaml
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
