sudo route add -net 172.20.0.0 netmask 255.255.0.0 gw 192.168.58.101

#docker build -t pasha.cafebazaar.ir:5000/colonelm_test:latest .
#docker push pasha.cafebazaar.ir:5000/colonelm_test:latest
rm blacksmith
(go generate && go build) || exit 1
DOCKER_IMAGE="192.168.58.1:5000/colonelm_test"
docker build -t 192.168.58.1:5000/colonelm_test:latest . || exit 1
docker push 192.168.58.1:5000/colonelm_test:latest || exit 1

ssh colonelmo@172.20.0.2 'rm -rf ~/deployed && mkdir ~/deployed'
scp -r . colonelmo@172.20.0.2:~/deployed/ &>/dev/null

ssh colonelmo@172.20.0.2 'bash ~/deployed/remote_deploy.sh'
