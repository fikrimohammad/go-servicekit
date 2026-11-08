// Package s3 wraps the AWS S3 SDK transfer manager, S3 client, and presign
// client and records OpenTelemetry traces and metrics for every operation.
//
// # When to use
//
// Use s3 when a service stores objects in S3 or an S3-compatible endpoint and
// needs multipart transfer and presigned URLs behind one client. It is not a
// filesystem abstraction; callers choose buckets, keys, and content types.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/s3"
//
// # Minimal example
//
//	import awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
//
//	client, err := s3.New(ctx, s3.Config{
//		Region:          "us-east-1",
//		Endpoint:        "http://localhost:9000",
//		AccessKeyID:     "minioadmin",
//		SecretAccessKey: "minioadmin",
//	})
//	if err != nil {
//		return err
//	}
//	_, err = client.PutObject(ctx, &awss3.PutObjectInput{
//		Bucket: aws.String("reports"), Key: aws.String("q4.txt"), Body: body,
//	})
//	if s3.IsNotFound(err) {
//		// the key does not exist
//	}
//
// # Lifecycle
//
// [New] builds one underlying S3 client and derives the transfer manager and
// presign client from it. The returned [Client] is safe for concurrent use and
// has no Close; build it once at startup and reuse it. [Config] requires
// Region; when Endpoint is set, path-style addressing is the default.
//
// # Common mistakes
//
//   - Leaving Region empty, or looking for a Close method.
//   - Forgetting UsePathStyle for an endpoint that needs path-style URLs.
//   - Creating a client per request instead of sharing it.
//
// # Test recipe
//
// Run gofakes3 in-process and point Config.Endpoint at it; the package's tests
// use the real New path, so no live S3 server is needed.
package s3
