#!/bin/sh

set -e

package_name="DockerDeploy"
package_version=$(git describe --tags)
arch="arm64"
maintainer="Iain MacDonald <IJMacD@gmail.com>"
homepage="https://github.com/IJMacD/docker-deploy"
description="Deploy software using Docker Compose files\n Specify a remote HTTP endpoint to serve Docker Compose files\n which will be automatically applied as they are updated."

./build.sh linux $arch

size=$(ls -l build/linux-$arch/docker-deploy | awk '{print int($5/1024)}')

out_dir="build/${package_name}-${package_version}"

rm -rf $out_dir
mkdir -p $out_dir/DEBIAN/
mkdir -p $out_dir/usr/bin

echo "Package: $package_name
Version: $package_version
Architecture: $arch
Maintainer: $maintainer
Installed-Size: 7562
Homepage: $homepage
Description: $description" > $out_dir/DEBIAN/control

cp -r package/* $out_dir

cp "build/linux-$arch/docker-deploy" "$out_dir/usr/bin/"

sudo chown -R 0:0 $out_dir
 
dpkg -b $out_dir