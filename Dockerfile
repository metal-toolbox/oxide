FROM alpine:3.8 AS stage1

# IPMITOOL

ARG IPMITOOL_REPO=https://github.com/ipmitool/ipmitool.git
ARG IPMITOOL_COMMIT=19d78782d795d0cf4ceefe655f616210c9143e62

WORKDIR /tmp
RUN apk add --update --upgrade --no-cache --virtual build-deps\
            alpine-sdk \
            automake \
            autoconf \
            libtool \
            openssl-dev \
            readline-dev \
    && git clone -b master ${IPMITOOL_REPO}

#
# cherry-pick'ed 1edb0e27e44196d1ebe449aba0b9be22d376bcb6
# to fix https://github.com/ipmitool/ipmitool/issues/377
#
WORKDIR /tmp/ipmitool
RUN git checkout ${IPMITOOL_COMMIT} \
    && git config --global user.email "github.ci@doesnot.existorg" \
    && git cherry-pick 1edb0e27e44196d1ebe449aba0b9be22d376bcb6 \
    && ./bootstrap \
    && ./configure \
        --prefix=/usr/local \
        --enable-ipmievd \
        --enable-ipmishell \
        --enable-intf-lan \
        --enable-intf-lanplus \
        --enable-intf-open \
    && make \
    && make install \
    && apk del build-deps

WORKDIR /tmp
RUN rm -rf /tmp/ipmitool

## Get IPMI IANA resource, to prevent dependency on third party servers at runtime.
WORKDIR /usr/share/misc
RUN wget https://www.iana.org/assignments/enterprise-numbers.txt

# Supermicro SUM
WORKDIR /tmp/sum

## Download
RUN wget https://www.supermicro.com/Bios/sw_download/698/sum_2.14.0_Linux_x86_64_20240215.tar.gz -O sum.tar.gz
RUN mkdir -p unzipped
RUN tar -xvzf sum.tar.gz -C unzipped --strip-components=1

## Install
RUN cp unzipped/sum /usr/bin/sum #TODO; smc sum has the same name as the gnu command sum (/usr/bin/sum). So we are overwritting it. Sorry not Sorry.
RUN chmod +x /usr/bin/sum

WORKDIR /tmp
RUN rm -rf /tmp/sum

# Build a lean image with dependencies installed.
## Do this because apk can install a ton of junk.
FROM alpine:3.8
COPY --from=stage1 / /

## SUM and IPMITOOL is dynamically linked and needs glibc
ENV GLIBC_REPO=https://github.com/sgerrand/alpine-pkg-glibc
ENV GLIBC_VERSION=2.30-r0
RUN set -ex && \
    apk --update add libstdc++ curl ca-certificates && \
    for pkg in glibc-${GLIBC_VERSION} glibc-bin-${GLIBC_VERSION}; \
        do curl -sSL ${GLIBC_REPO}/releases/download/${GLIBC_VERSION}/${pkg}.apk -o /tmp/${pkg}.apk; done && \
    apk add --allow-untrusted /tmp/*.apk && \
    rm -v /tmp/*.apk && \
    /usr/glibc-compat/sbin/ldconfig /lib /usr/glibc-compat/lib

# required by ipmitool and sum runtime
RUN apk add --update --upgrade --no-cache --virtual run-deps \
        ca-certificates \
        libcrypto1.0 \
        readline

COPY bioscfg /usr/sbin/bioscfg
RUN chmod +x /usr/sbin/bioscfg

ENTRYPOINT ["/usr/sbin/bioscfg"]