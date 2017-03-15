#!/bin/bash
set -e


die() {
  echo
  echo "$@" 1>&2;
  exit 1
}

# Variables are injected from this configuration file:

cert_dir='./certs'

if [ ! -d ${cert_dir} ]; then
    mkdir -p ${cert_dir}
else
    cp -rf ${cert_dir} "${cert_dir}-bak-`date +%s`"
fi

sans="IP:127.0.0.1,DNS:localhost"

rm -r easy-rsa-master || echo "Not Fatal"
wget -nc https://storage.googleapis.com/kubernetes-release/easy-rsa/easy-rsa.tar.gz || die "Failed while downloading the easy-rsa"
tar xzf easy-rsa.tar.gz
rm easy-rsa.tar.gz

easyrsa3_dir=easy-rsa-master/easyrsa3


cd $easyrsa3_dir
./easyrsa init-pki

# CA
./easyrsa --batch "--req-cn=127.0.0.1@`date +%s`" build-ca nopass

# Master
./easyrsa --subject-alt-name="${sans}" build-server-full server nopass

# Admin
./easyrsa build-client-full client nopass
cd -


# Export certificates 
cp -p $easyrsa3_dir/pki/ca.crt ${cert_dir}/
cp -p $easyrsa3_dir/pki/private/ca.key ${cert_dir}/

cp -p $easyrsa3_dir/pki/issued/server.crt "${cert_dir}/"
cp -p $easyrsa3_dir/pki/private/server.key "${cert_dir}/"

cp -p $easyrsa3_dir/pki/issued/client.crt "${cert_dir}/"
cp -p $easyrsa3_dir/pki/private/client.key "${cert_dir}/"


CLIENT_CERT_PEM=$(openssl x509 -in $easyrsa3_dir/pki/issued/client.crt)
CLIENT_KEY_PEM=$(cat $easyrsa3_dir/pki/private/client.key)
export CLIENT_PKCS12_PASSWORD=$@
echo -e "$CLIENT_KEY_PEM\n$CLIENT_CERT_PEM" | openssl pkcs12 -export -password env:CLIENT_PKCS12_PASSWORD -name "Client Keys" -out ${cert_dir}/client.pfx
unset CLIENT_PKCS12_PASSWORD
