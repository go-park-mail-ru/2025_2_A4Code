FROM tianon/exim4

# install exim4-daemon-heavy to get DKIM support
RUN apt-get update && \
    apt-get install -y --no-install-recommends exim4-daemon-heavy && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

COPY exim.conf /etc/exim4/exim4.conf
RUN chown root:Debian-exim /etc/exim4/exim4.conf && chmod 640 /etc/exim4/exim4.conf
