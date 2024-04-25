FROM registry.access.redhat.com/ubi9/python-39:5b8dd87d19a7f9e5ba0b900471f0920e109e0412fbdc49539073ed86afdadf8a

# Pin versions in pip.
# hadolint ignore=DL3013
RUN pip install --no-cache-dir mkdocs mkdocs-material markdown-include

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
