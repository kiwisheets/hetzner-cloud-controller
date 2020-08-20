#!/bin/sh
export DEBIAN_FRONTEND=noninteractive
apt update
apt full-upgrade -y
apt install -y docker.io docker-compose git
systemctl enable --now docker.service

# clone your repo and start
# git clone https://github.com/kiwisheets/kiwisheets --recursive
# cd ./kiwisheets/deployconfig
# docker stack deploy -c production-deploy.yml kiwisheets

touch /root/cloud-init-was-here
reboot
