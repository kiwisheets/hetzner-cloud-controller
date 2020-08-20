#!/bin/sh
export DEBIAN_FRONTEND=noninteractive
apt update
apt full-upgrade -y
apt install -y docker.io docker-compose git
systemctl enable --now docker.service

# it is highly recommended to run docker swarm mode on an internal network
# tokens will be stored in root directory
docker swarm init --advertise-addr ens10 --listen-addr ens10 > docker_swarm.txt
docker swarm join-token manager > docker_swarm_manager.txt

touch /root/cloud-init-was-here
reboot
