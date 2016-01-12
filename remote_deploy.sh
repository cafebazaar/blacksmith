#sudo docker stop etcd
#sudo docker stop aghajoon_docker
#docker pull pasha.cafebazaar.ir:5000/colonelm_test:latest
docker pull 192.168.58.1:5000/colonelm_test:latest

sudo docker run -d -p 4001:4001 -p 2380:2380 -p 2379:2379 --restart=always --name etcd quay.io/coreos/etcd:v2.0.3  -name etcd0  -advertise-client-urls http://172.20.0.2:2379,http://172.20.0.2:4001  -listen-client-urls http://0.0.0.0:2379,http://0.0.0.0:4001  -initial-advertise-peer-urls http://172.20.0.2:2380  -listen-peer-urls http://0.0.0.0:2380  -initial-cluster-token etcd-cluster-1  -initial-cluster etcd0=http://172.20.0.2:2380  -initial-cluster-state new -cors '*'


sudo DOCKER_IMAGE="192.168.58.1:5000/colonelm_test" ~/deployed/install-as-docker.sh ~/aghajoon-workspace-kubernetes/workspace http://172.20.0.2:4001 enp0s8
