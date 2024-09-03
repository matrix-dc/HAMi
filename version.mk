GO=go
GO111MODULE=on
CMDS=scheduler vGPUmonitor
DEVICES=nvidia
OUTPUT_DIR=bin
TARGET_ARCH=amd64
# 镜像在咱们的仓库中有 切记从docker hub拉去的时候要加入--platform=linux/amd64
GOLANG_IMAGE=golang:1.21-bullseye
NVIDIA_IMAGE=nvidia/cuda:12.2.0-devel-ubuntu20.04
DEST_DIR=/usr/local/vgpu/

VERSION = v2.3.12-20240903-c278112
IMG_NAME =hami
IMG_TAG="${IMG_NAME}:${VERSION}"