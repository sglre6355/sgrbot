FROM rust:slim AS build-env
RUN apt-get update && apt-get install -y cmake
WORKDIR /app
RUN --mount=type=bind,source=src,target=src \
    --mount=type=bind,source=Cargo.toml,target=Cargo.toml \
    --mount=type=bind,source=Cargo.lock,target=Cargo.lock \
    --mount=type=cache,target=/app/target/ \
    --mount=type=cache,target=/usr/local/cargo/registry/ \
    cargo build --locked --release && \
    cp /app/target/release/sgrbot /app/sgrbot

FROM gcr.io/distroless/cc
COPY --from=build-env /app/sgrbot /
CMD ["./sgrbot"]
