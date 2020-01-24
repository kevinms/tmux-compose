#!/bin/bash

set -e

RELEASE=$1
if [[ "$RELEASE" == "" ]]; then
	echo "Usage: $0 <release>"
	exit 1
fi

OS=("darwin" "linux")
ARCH=("386" "amd64")

for os in "${OS[@]}"; do
	for arch in "${ARCH[@]}"; do
		echo -n "Build $os $arch "
		GOOS=$os GOARCH=$arch go build
		zip tmux-compose-$RELEASE-$os-$arch.zip tmux-compose
		rm tmux-compose
	done
done
