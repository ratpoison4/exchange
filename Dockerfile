FROM alpine:latest
MAINTAINER Alexander Zaytsev "thebestzorro@yandex.ru"
RUN apk update && \
    apk upgrade && \
    apk add ca-certificates tzdata
ADD exchange /bin/exchange
RUN chmod 0755 /bin/exchange
EXPOSE 8070
VOLUME ["/data/conf/"]
ENTRYPOINT ["exchange"]
CMD ["-config", "/data/conf/exchange.json"]