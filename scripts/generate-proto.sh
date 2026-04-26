#!/bin/bash
set -e

export PATH="$PATH:$HOME/go/bin"

rm -rf proto/generated
mkdir -p proto/generated/v1 proto/generated/internal

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

protoc \
    --go_out=$TMPDIR \
    --go_opt=paths=source_relative \
    -I. -Iproto \
    proto/coco/v1/coco.proto \
    proto/coco/v1/checkpoint.proto \
    proto/coco/v1/node.proto \
    proto/coco/v1/master.proto \
    proto/coco/v1/cluster.proto \
    proto/coco/v1/network.proto \
    proto/internal/visor.proto \
    proto/internal/agent.proto

protoc \
    --connect-go_out=$TMPDIR \
    --connect-go_opt=paths=source_relative \
    -I. -Iproto \
    proto/coco/v1/coco.proto \
    proto/coco/v1/checkpoint.proto \
    proto/coco/v1/node.proto \
    proto/coco/v1/master.proto \
    proto/coco/v1/cluster.proto \
    proto/coco/v1/network.proto

cp -r $TMPDIR/proto/coco/v1/* proto/generated/v1/
cp -r $TMPDIR/proto/internal/* proto/generated/internal/

echo "Generated:"
ls proto/generated/v1/
ls proto/generated/internal/
