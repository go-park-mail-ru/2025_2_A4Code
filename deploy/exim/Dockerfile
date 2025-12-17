FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN rm -f /etc/apt/sources.list.d/debian.sources && \
    echo "deb http://mirror.yandex.ru/debian bookworm main" > /etc/apt/sources.list && \
    echo "deb http://mirror.yandex.ru/debian bookworm-updates main" >> /etc/apt/sources.list && \
    echo "deb http://mirror.yandex.ru/debian-security bookworm-security main" >> /etc/apt/sources.list

# install exim4-daemon-heavy to get DKIM support
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt \
    rm -f /var/lib/apt/lists/lock /var/cache/apt/archives/lock /var/lib/dpkg/lock-frontend && \
    apt-get -o Acquire::http::Timeout=20 -o Acquire::Retries=5 update && \
    apt-get -o Acquire::http::Timeout=20 -o Acquire::Retries=5 install -y --no-install-recommends exim4-daemon-heavy ca-certificates && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# our config
COPY exim.conf /etc/exim4/exim4.conf
RUN chown root:Debian-exim /etc/exim4/exim4.conf && chmod 640 /etc/exim4/exim4.conf

CMD ["exim", "-bdf", "-q30m"]
