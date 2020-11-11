FROM golang:1.14

RUN apt update && \
    apt install -y mesa-opencl-icd ocl-icd-opencl-dev gcc git bzr jq pkg-config curl

ENV WORKDIR=/go-epik
WORKDIR ${WORKDIR}