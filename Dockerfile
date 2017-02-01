FROM quay.io/brianredbeard/corebox

ENTRYPOINT ["/app/blacksmith"]
WORKDIR /app

COPY blacksmith /app/
