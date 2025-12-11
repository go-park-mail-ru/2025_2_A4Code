FROM tianon/exim4

ENV DEBIAN_FRONTEND=noninteractive

# install exim4-daemon-heavy to get DKIM support (используем одно зеркало + ретраи)
RUN set -e; \
    echo 'deb http://mirror.yandex.ru/debian trixie main' > /etc/apt/sources.list && \
    echo 'deb http://mirror.yandex.ru/debian trixie-updates main' >> /etc/apt/sources.list && \
    echo 'Acquire::Retries "5"; Acquire::http::Timeout "20"; Acquire::https::Timeout "20";' > /etc/apt/apt.conf.d/99retries && \
    apt-get update && \
    apt-get install -y --no-install-recommends exim4-daemon-heavy && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

COPY exim.conf /etc/exim4/exim4.conf
RUN chown root:Debian-exim /etc/exim4/exim4.conf && chmod 640 /etc/exim4/exim4.conf
