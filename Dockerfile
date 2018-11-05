FROM alpine
WORKDIR /app/
ADD sytral-rt .
HEALTHCHECK --interval=10s --timeout=3s CMD curl -f http://localhost:8080/status || exit 1
ENV GIN_MODE=release
ENV PORT=8080
ENV SYTRALRT_JSON_LOG=1
ENV SYTRALRT_LOG_LEVEL=info
CMD ["./sytral-rt"]
