FROM --platform=linux/amd64 golang:1.16

RUN sed -i "s@http://deb.debian.org@http://mirrors.aliyun.com@g" /etc/apt/sources.list && rm -Rf /var/lib/apt/lists/* && apt-get update

RUN apt-get update && apt-get install -y ca-certificates build-essential clang ocl-icd-opencl-dev ocl-icd-libopencl1 jq libhwloc-dev

# ARG RUST_VERSION=nightly
# ENV XDG_CACHE_HOME="/tmp"

# ENV RUSTUP_HOME=/usr/local/rustup \
#     CARGO_HOME=/usr/local/cargo \
#     PATH=/usr/local/cargo/bin:$PATH

# RUN wget "https://static.rust-lang.org/rustup/dist/x86_64-unknown-linux-gnu/rustup-init"; \
#     chmod +x rustup-init; \
#     ./rustup-init -y --no-modify-path --profile minimal --default-toolchain $RUST_VERSION; \
#     rm rustup-init; \
#     chmod -R a+w $RUSTUP_HOME $CARGO_HOME; \
#     rustup --version; \
#     cargo --version; \
#     rustc --version;

COPY ./ /opt/epik
WORKDIR /opt/epik

RUN git submodule update --init --recursive

ENV GOPROXY=https://goproxy.cn
RUN go mod tidy
RUN make all
RUN make install

ENV IPFS_GATEWAY="https://proof-parameters.s3.cn-south-1.jdcloud-oss.com/ipfs/"
RUN epik fetch-params 8MiB

ARG FULLNODE_API_INFO
ENV FULLNODE_API_INFO=$FULLNODE_API_INFO

ARG COINBASE
ENV COINBASE=${COINBASE}

COPY ./docker/scripts/setup.sh /opt/epik/setup.sh
