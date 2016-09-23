FROM alpine
ADD bin/depman-srv-static /depman-srv
ADD docker-entrypoint.sh /docker-entrypoint.sh
EXPOSE 8082
ENV LISTEN="0.0.0.0:8082" \
    LOGLEVEL="debug" \
    STOREDIR="/depman_data" \
    NAMESPACE="master"

VOLUME /tmp/depman_files

ENTRYPOINT [ "/docker-entrypoint.sh" ]
CMD [ "depman-srv" ]
