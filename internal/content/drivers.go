package content

import (
	// Blob storage drivers for go-cloud
	// These blank imports register the URL openers for each provider.
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/memblob"
	_ "gocloud.dev/blob/s3blob"
)
