#!/bin/bash
COREOS_CHANNEL=$1
COREOS_VERSION=$2
DIR=fs/${COREOS_VERSION}

die() {
  echo
  echo "$@" 1>&2;
  exit 1
}

if [ -e ${DIR} ]; then
  echo "${DIR} directory exists"
  exit 0
fi

mkdir -p ${DIR}/
cd ${DIR}

command -v gpg >/dev/null 2>&1 || { echo >&2 "this script require The GNU Privacy Guard(gpg) but it's not installed.  Aborting."; exit 1; }
# adding the coreOS image signing key for verifications
gpg --import --keyid-format LONG CoreOS_Image_Signing_Key.asc
wget -Nc http://${COREOS_CHANNEL}.release.core-os.net/amd64-usr/${COREOS_VERSION}/coreos_production_pxe.vmlinuz || die "Failed while downloading the kernel image"
wget -Nc http://${COREOS_CHANNEL}.release.core-os.net/amd64-usr/${COREOS_VERSION}/coreos_production_pxe.vmlinuz.sig || die "Failed while downloading the signature of the kernel image"
gpg --verify coreos_production_pxe.vmlinuz.sig || die "The downloaded kernel image is corrupted"
wget -Nc http://${COREOS_CHANNEL}.release.core-os.net/amd64-usr/${COREOS_VERSION}/coreos_production_pxe_image.cpio.gz || die "Failed while downloading the initrd image"
wget -Nc http://${COREOS_CHANNEL}.release.core-os.net/amd64-usr/${COREOS_VERSION}/coreos_production_pxe_image.cpio.gz.sig || die "Failed while downloading the signature of the initrd image"
gpg --verify coreos_production_pxe_image.cpio.gz.sig || die "The downloaded initrd image is corrupted"
wget -Nc http://${COREOS_CHANNEL}.release.core-os.net/amd64-usr/${COREOS_VERSION}/coreos_production_image.bin.bz2 || die "Failed while downloading the installation image"
wget -Nc http://${COREOS_CHANNEL}.release.core-os.net/amd64-usr/${COREOS_VERSION}/coreos_production_image.bin.bz2.sig || die "Failed while downloading the signature of the installation image"
gpg --verify coreos_production_image.bin.bz2.sig || die "The downloaded kernel image is corrupted"

cd -
