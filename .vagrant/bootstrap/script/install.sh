#!/usr/bin/env bash
echo -e
echo -e "----------------------------------------"
echo -e "---------- Updating packages list ------\n"

yum -y update

echo -e
echo -e "----------------------------------------"
echo -e "---------- Installing vim --------------\n"

yum -y install vim

echo -e
echo -e "---------- Installing wget -------------\n"

yum -y install wget

echo -e
echo -e "---------- Installing git --------------\n"

yum -y install git

echo -e
echo -e "---------- Installing htop -------------\n"

yum -y install epel-release
yum -y install htop

echo -e
echo -e "----------------------------------------"
echo -e "---------- Installing golang -----------\n"

GO_VERSION="1.12.6"

/usr/bin/wget -q -O /usr/local/src/go${GO_VERSION}.linux-amd64.tar.gz \
    https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz

/bin/tar -C /usr/local -xzf /usr/local/src/go${GO_VERSION}.linux-amd64.tar.gz

echo -e
echo -e "----------------------------------------"
echo -e "---------- Installing Percona ----------\n"

yum -y install 'https://repo.percona.com/yum/percona-release-latest.noarch.rpm'
percona-release setup ps57
yum -y install Percona-Server-server-57 Percona-Server-devel-57

echo -e
echo -e "----------------------------------------"
echo -e "---------- Installing Redis ------------\n"

yum -y install gcc make tcl
REDIS_VER=4.0.12
wget http://download.redis.io/releases/redis-$REDIS_VER.tar.gz
tar xzvf redis-$REDIS_VER.tar.gz
rm redis-$REDIS_VER.tar.gz
cd redis-$REDIS_VER/deps
make hiredis lua jemalloc linenoise
make geohash-int
cd ../
make
make install
cd utils

PORT=6379
CONFIG_FILE=/etc/redis/6379.conf
LOG_FILE=/var/log/redis_6379.log
DATA_DIR=/var/lib/redis/6379
EXECUTABLE=/usr/local/bin/redis-server

echo -e \
  "${PORT}\n${CONFIG_FILE}\n${LOG_FILE}\n${DATA_DIR}\n${EXECUTABLE}\n" | \
  ./install_server.sh

cd /usr/local/bin && ln -s redis-cli redis

systemctl start redis_6379
