#!/bin/sh
export DEBIAN_FRONTEND=noninteractive
apt update
apt full-upgrade -y
apt install -y docker.io docker-compose git
systemctl enable --now docker.service

# it is highly recommended to run docker swarm mode on an internal network
# log will be stored in root directory
docker swarm join --advertise-addr ens10 --listen-addr ens10 --token ***YOUR SWARM TOKEN*** > docker_swarm.txt

touch /root/cloud-init-was-here
reboot
