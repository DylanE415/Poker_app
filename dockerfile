# Runtime-only image, uses your prebuilt binary
FROM gcr.io/distroless/static-debian12

# Everything will live under /app in the container
WORKDIR /app

# 1) Copy your prebuilt Go binary
#    (must be a Linux binary: GOOS=linux GOARCH=amd64 CGO_ENABLED=0)
COPY build/server/server ./server/server

# 2) Copy your static assets
COPY build/static ./static

# 3) Copy your users directory
COPY build/users  ./users

# Your server should listen on 8080 inside the container
ENV PORT=8080
EXPOSE 8080

# Run the server
ENTRYPOINT ["/app/server/server"]
