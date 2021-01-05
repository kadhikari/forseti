FROM alpine
WORKDIR /app/
RUN  apk add --no-cache curl tzdata
ADD forseti .
HEALTHCHECK --interval=10s --timeout=3s CMD curl -f http://localhost:8080/status || exit 1
ENV GIN_MODE=release
ENV PORT=8080
ENV FORSETI_JSON_LOG=1
ENV FORSETI_LOG_LEVEL=info
CMD ["./forseti"]
