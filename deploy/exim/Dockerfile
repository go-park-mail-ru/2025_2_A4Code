FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# install exim4-daemon-heavy to get DKIM support
RUN apt-get update && \
    apt-get install -y --no-install-recommends exim4-daemon-heavy ca-certificates && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# our config
COPY exim.conf /etc/exim4/exim4.conf
RUN chown root:Debian-exim /etc/exim4/exim4.conf && chmod 640 /etc/exim4/exim4.conf

CMD ["exim", "-bdf", "-q30m"]
