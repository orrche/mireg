FROM fedora

COPY docker_entrypoint.sh /docker_entrypoint.sh
COPY mireg /mireg
RUN mkdir /data

EXPOSE 5000
WORKDIR /data

CMD ["/docker_entrypoint.sh"]
