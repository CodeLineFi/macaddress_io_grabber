#!/usr/bin/env bash
export CONFIG_DIR=$1
export DOCUMENT_ROOT=$2
export DB_PASSWORD=$3
export DB_NAME=$4

echo -e "\n--------- Configuring MySQL ---------\n"

systemctl start mysqld

mysqlpwd=$(cat /var/log/mysqld.log | grep "A temporary password is generated for" | awk '{print $NF}')

mysql -uroot -p"$mysqlpwd" --connect-expired-password -e \
    "SET PASSWORD = PASSWORD('${DB_PASSWORD}');"

mysql -u root -p"${DB_PASSWORD}" -e \
    "CREATE DATABASE IF NOT EXISTS ${DB_NAME};"

systemctl stop mysqld
systemctl start mysqld

echo -e "\n--------- Configuring Golang --------\n"

echo ". ${CONFIG_DIR}/goenv.sh" >> /etc/bashrc

echo -e "\n--------- Configuring Project -------\n"
echo -e `pwd`
echo -e ${CONFIG_DIR}

sudo -u vagrant cp ${CONFIG_DIR}/.my.cnf /home/vagrant/.my.cnf
sudo -u vagrant cp -n ${DOCUMENT_ROOT}/config.example.json ${DOCUMENT_ROOT}/config.json

