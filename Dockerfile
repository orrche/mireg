FROM fedora

COPY mireg /mireg
RUN mkdir /data

EXPOSE 5000
WORKDIR /data


CMD ["/mireg"]
