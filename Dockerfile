FROM alpine:latest

COPY bioscfg /usr/sbin/bioscfg
RUN chmod +x /usr/sbin/bioscfg

ENTRYPOINT ["/usr/sbin/bioscfg"]