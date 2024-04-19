FROM alpine:3.8 as runner

COPY bioscfg /usr/sbin/bioscfg
RUN chmod +x /usr/sbin/bioscfg

ENTRYPOINT bioscfg
