FROM quay.io/brianredbeard/corebox

MAINTAINER Reza Mohammadi "<reza@cafebazaar.ir>"
MAINTAINER Mohammad Nasirifar <far.nasiri.m@gmail.com> @colonelmo

ENTRYPOINT ["/app/blacksmith"]
WORKDIR /app

COPY blacksmith /app/
