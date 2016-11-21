FROM ubuntu:16.10

RUN apt-get update && apt-get install -y --no-install-recommends libgit2-24

ENTRYPOINT ["/app/blacksmith"]
WORKDIR /app

COPY blacksmith /app/
