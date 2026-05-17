FROM gcr.io/distroless/static-debian13:nonroot

COPY dynamic-markdown-site /app/dynamic-markdown-site

EXPOSE 8080

USER 65532:65532

ENV PORT=8080 \
    DYNAMIC_MARKDOWN_PORT=8080 \
    DYNAMIC_MARKDOWN_LOG_LEVEL=info \
    DYNAMIC_MARKDOWN_CACHE=true \
    DYNAMIC_MARKDOWN_ROOT=/content

VOLUME ["/content"]

ENTRYPOINT ["/app/dynamic-markdown-site"]
CMD ["-root", "/content", "-port", "8080", "-cache"]
