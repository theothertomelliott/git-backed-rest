# Git Backed REST

Git is great for storing code, and maybe the occasional image or two.
It also makes it easy for teams to collaborate and deal with fine-grained changes.
But could it be used as a more arbitrary data store?
Can we force the square peg of Git into the round hole of a REST backing store?

This repository explores a few different approaches to this problem, with comparisons to non-Git
storage options (such as S3).

## The API

At its most basic a REST API allows you to manage resources with HTTP verbs, so I did just that.
To keep things simple, I'm building an API that interacts with byte slices at arbitrary paths using the following verbs:

* GET
* POST
* PUT
* DELETE

Note that `PATCH` and `OPTIONS` are out of scope since they require more complex interactions with
resources and their metadata.

### Example Usage

For instance, you could manage a user profile stored as JSON:

```bash
# Create a new user profile
POST /users/alice/profile
Content-Type: application/json
{"name": "Alice Smith", "email": "alice@example.com", "role": "developer"}
→ 201 Created

# Retrieve the profile
GET /users/alice/profile
→ 200 OK
→ {"name": "Alice Smith", "email": "alice@example.com", "role": "developer"}

# Update the profile
PUT /users/alice/profile
Content-Type: application/json
{"name": "Alice Smith", "email": "alice@newdomain.com", "role": "senior developer"}
→ 204 No Content

# Delete the profile
DELETE /users/alice/profile
→ 204 No Content
```

The path structure is arbitrary - you could use `/api/v1/organizations/acme/projects/website/config.json` or any other hierarchical structure that suits your needs.

This provides a very generic API that could be layered under middleware to provide
more focused APIs for specific use cases.

## Backends

Backends for the API can be provided by implementing the `APIBackend` interface defined in [api.go](api.go).

A few backends are currently implemented, for Git and other alternatives. These are all in packages under
the `backends` directory.

### Git Porcelain

A naive implementation of the interface using the Git CLI directly, specifically the porcelain commands
that a typical developer would use.

### Memory

An in-memory implementation of the interface, storing resources in a map.

### S3

An implementation of the interface using the AWS S3 SDK. This is compatible with other object storage providers,
such as Cloudflare's R2.

## Testing

Each backend provides unit (or integration) tests that can be run with `go test`.

Test for backends using third-party platforms may require credentials, which can be
set in a `.env` file.

See [.env.example](.env.example) for details of variables required for each
third-party platform.

Once configured, all tests may be run together:

```bash
go test ./...
```

Or you can run tests for a specific backend:

```bash
go test ./backends/gitporcelain
```

The `gitporcelain` and `s3` backends also include a script to generate a trace
of a test run, which allows comparison of the duration of different operations.

These scripts create the test file and then open a trace viewer. For example:

```bash
backends/gitporcelain/trace_test.sh
```