FROM debian:12.5-slim AS stage1

# Supermicro SUM
# Note: If we remove the SUM tool, we can move back to an alpine image. Then also compile bioscfg with CGO_ENABLED=0
WORKDIR /tmp/sum

## Pre-reqs
RUN apt-get update
RUN apt-get install wget -y

## Download
RUN wget https://www.supermicro.com/Bios/sw_download/698/sum_2.14.0_Linux_x86_64_20240215.tar.gz -O sum.tar.gz
RUN mkdir -p unzipped
RUN tar -xvzf sum.tar.gz -C unzipped --strip-components=1

## Install
RUN cp unzipped/sum /usr/sbin/sum #TODO; smc sum has the same name as the gnu command sum (/usr/bin/sum). So we are overwritting it. Sorry not Sorry.
RUN chmod +x /usr/sbin/sum

# IPMI Tool
#
# cherry-pick'ed 1edb0e27e44196d1ebe449aba0b9be22d376bcb6
# to fix https://github.com/ipmitool/ipmitool/issues/377
#
ARG IPMITOOL_REPO=https://github.com/ipmitool/ipmitool.git
ARG IPMITOOL_COMMIT=19d78782d795d0cf4ceefe655f616210c9143e62
ARG IPMITOOL_CHERRY_PICK=1edb0e27e44196d1ebe449aba0b9be22d376bcb6
ARG IPMITOOL_BUILD_DEPENDENCIES="git curl make autoconf automake libtool libreadline-dev"

WORKDIR /tmp/ipmi

## Pre-reqs
RUN apt-get update
RUN apt-get install ${IPMITOOL_BUILD_DEPENDENCIES} -y

## Download
RUN git clone -b master ${IPMITOOL_REPO}
WORKDIR /tmp/ipmi/ipmitool
RUN git checkout ${IPMITOOL_COMMIT}
RUN git config --global user.email "github.ci@doesnot.existorg"
RUN git cherry-pick ${IPMITOOL_CHERRY_PICK}

## Install
RUN ./bootstrap
RUN autoreconf -i
RUN ./configure \
    --prefix=/usr/local \
    --enable-ipmievd \
    --enable-ipmishell \
    --enable-intf-lan \
    --enable-intf-lanplus \
    --enable-intf-open
RUN make
RUN make install

# Build a lean image with dependencies installed.
FROM debian:12.5-slim
COPY --from=stage1 /usr/sbin/sum /usr/bin/sum
COPY --from=stage1 /usr/local/bin/ipmitool /usr/local/bin/ipmitool

## Install runtime dependencies
RUN apt-get update -y
RUN apt-get install libreadline8 --no-install-recommends -y

# BiosCfg

COPY bioscfg /usr/sbin/bioscfg
RUN chmod +x /usr/sbin/bioscfg

ENTRYPOINT ["/usr/sbin/bioscfg"]