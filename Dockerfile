FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    ca-certificates \
    fuse \
    libfuse-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY jaft .

EXPOSE 42567

CMD ["./jaft"]
