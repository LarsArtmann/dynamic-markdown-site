export const heroCode = `# Serve a local directory
dynamic-markdown-site -dev -root ./docs

# Or serve from S3
dynamic-markdown-site -storage-url s3://my-bucket/docs

# Or from Google Cloud Storage
dynamic-markdown-site -storage-url gs://my-bucket/docs`;
