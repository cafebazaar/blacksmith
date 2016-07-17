FROM quay.io/brianredbeard/corebox

MAINTAINER Reza Mohammadi "<reza@cafebazaar.ir>"

ENTRYPOINT ["/app/blacksmith"]
WORKDIR /app

COPY blacksmith /app/
