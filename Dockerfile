FROM alpine:latest

ENTRYPOINT ["/usr/sbin/bioscfg"]

COPY bioscfg /usr/sbin/bioscfg
RUN chmod +x /usr/sbin/bioscfg
