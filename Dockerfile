
FROM busybox:ubuntu-14.04

MAINTAINER Reza Mohammadi "<reza@cafebazaar.ir>"

ENTRYPOINT ["/app/blacksmith"]
WORKDIR /app

COPY blacksmith /app/
COPY . /app/
