FROM rust:slim AS build-env
RUN apt-get update && apt-get install -y cmake
WORKDIR /app
COPY . /app
RUN cargo build --release

FROM gcr.io/distroless/cc
COPY --from=build-env /app/target/release/sgrbot /
CMD ["./sgrbot"]
