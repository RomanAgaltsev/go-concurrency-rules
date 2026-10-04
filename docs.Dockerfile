# syntax=docker/dockerfile:1

# The site's builder: Zensical, pinned in requirements-docs.txt, for hosts
# without Python. `task docs:build` and `task docs:serve` run it with the
# repository mounted at /site. CI does not use this image — it installs the
# same requirements-docs.txt with actions/setup-python.
FROM python:3.13.16-slim AS docs

LABEL org.opencontainers.image.source="https://github.com/RomanAgaltsev/go-concurrency-rules" \
      org.opencontainers.image.description="Zensical, pinned, for building the go-concurrency-rules site"

# The requirements are bind-mounted, not copied: they never enter a layer, and
# the pip cache lives in a cache mount, outside the image.
RUN --mount=type=bind,source=requirements-docs.txt,target=/tmp/requirements-docs.txt \
    --mount=type=cache,target=/root/.cache/pip \
    pip install --root-user-action=ignore --no-compile -r /tmp/requirements-docs.txt

RUN groupadd --system --gid 1001 docs && \
    useradd --system --uid 1001 --gid docs --no-create-home docs

WORKDIR /site
USER 1001:1001
EXPOSE 8000

ENTRYPOINT ["zensical"]
# --strict: without it a broken link is only a warning (spec probe P18).
CMD ["build", "--strict"]
