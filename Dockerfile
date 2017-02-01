FROM scratch

ENTRYPOINT ["/app/blacksmith"]
WORKDIR /app

COPY blacksmith /app/
