# syntax=docker.io/docker/dockerfile:1.24.0@sha256:87999aa3d42bdc6bea60565083ee17e86d1f3339802f543c0d03998580f9cb89
ARG OCI_SOURCE=https://github.com/zgiai/luas
ARG OCI_REVISION=unknown
ARG OCI_VERSION=dev

# Keep native dependencies ABI-compatible between build and runtime.
FROM docker.io/library/node:22.23.1-alpine3.24@sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2 AS base

# Install dependencies only when needed
FROM base AS deps
RUN apk add --no-cache libc6-compat
WORKDIR /app

# Install dependencies based on the preferred package manager
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN corepack enable pnpm && pnpm i --frozen-lockfile

# Rebuild the source code only when needed
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN mkdir -p public

# Disable anonymous Next.js telemetry in the build and production runtime.
ENV NEXT_TELEMETRY_DISABLED=1

RUN corepack enable pnpm && pnpm run build

# Materialize a runtime root without development package managers. The scratch
# copy prevents removed npm/Corepack/Yarn files from surviving in older layers.
FROM base AS runtime-root
RUN rm -rf /usr/local/lib/node_modules /usr/local/include \
        /usr/local/share/doc /usr/local/share/man /opt/yarn-* \
    && rm -f /usr/local/bin/corepack /usr/local/bin/npm /usr/local/bin/npx \
        /usr/local/bin/pnpm /usr/local/bin/pnpx /usr/local/bin/yarn \
        /usr/local/bin/yarnpkg /usr/local/bin/docker-entrypoint.sh /sbin/apk

# Production image, copy only the cleaned runtime and standalone application.
FROM scratch AS runner

ARG OCI_SOURCE
ARG OCI_REVISION
ARG OCI_VERSION

LABEL org.opencontainers.image.title="Luas Web" \
    org.opencontainers.image.description="Production runtime for the Luas Next.js scaffold" \
    org.opencontainers.image.source="${OCI_SOURCE}" \
    org.opencontainers.image.revision="${OCI_REVISION}" \
    org.opencontainers.image.version="${OCI_VERSION}" \
    org.opencontainers.image.base.name="docker.io/library/node:22.23.1-alpine3.24" \
    org.opencontainers.image.base.digest="sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2"

COPY --from=runtime-root / /

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

COPY --from=builder --chown=1000:1000 /app/public ./public

# Automatically leverage output traces to reduce image size
# https://nextjs.org/docs/advanced-features/output-file-tracing
COPY --from=builder --chown=1000:1000 /app/.next/standalone ./
COPY --from=builder --chown=1000:1000 /app/.next/static ./.next/static

USER 1000:1000

EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD ["node", "-e", "fetch(`http://127.0.0.1:${process.env.PORT || '3000'}/robots.txt`).then((response) => process.exit(response.ok ? 0 : 1)).catch(() => process.exit(1))"]

# server.js is created by next build from the standalone output
# https://nextjs.org/docs/pages/api-reference/next-config-js/output
CMD ["node", "server.js"]
