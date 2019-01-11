FROM alpine:3.8
MAINTAINER myonlyzzy@gmai.com
ARG TZ="Asia/Shanghai"
ENV TZ ${TZ}
COPY cmd/promethes-operator/promethes-operator /usr/local/bin/prometheus-operator
RUN chmod +x /usr/local/bin/prometheus-operator
CMD ["/usr/local/bin/prometheus-operator"]
