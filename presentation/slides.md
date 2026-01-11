---
# try also 'default' to start simple
theme: default
background: images/pexels-element5-1370294.jpg
# some information about your slides (markdown enabled)
title: Can Git Replace S3?
info: |
  ## Slidev Starter Template
  Presentation slides for developers.

  Learn more at [Sli.dev](https://sli.dev)
# apply UnoCSS classes to the current slide
class: text-center
# enable MDC Syntax: https://sli.dev/features/mdc
mdc: true
---

# Can you use Git to replace S3?

And more importantly *should* you?

Tom Elliott - telliott.me

<div class="abs-br m-6 text-xl">
  <button @click="$slidev.nav.openInEditor()" title="Open in Editor" class="slidev-icon-btn">
    <carbon:edit />
  </button>
  <a href="https://github.com/slidevjs/slidev" target="_blank" class="slidev-icon-btn">
    <carbon:logo-github />
  </a>
</div>

<!--
The last comment block of each slide will be treated as slide notes. It will be visible and editable in Presenter Mode along with the slide. [Read more in the docs](https://sli.dev/guide/syntax.html#notes)
-->

---

# About me

<!--
Hi everyone, I'm Tom, I've been using Go for the past decade, greatest longevity I've got out of any language.

I recently closed down a startup I was building out, so I've been tearing down all the tech I built for parts, blog posts and talks.

Today I'll be talking to you about one of those projects.
-->

<QRCode
    :width="300"
    :height="300"
    type="svg"
    data="https://telliott.me"
    :margin="10"
    :imageOptions="{ margin: 10 }"
/>

---

# What we're talking about

* Screenshots of the blog series

<QRCode
    :width="300"
    :height="300"
    type="svg"
    data="https://open.substack.com/pub/thefridaydeploy/p/can-git-back-a-rest-api-part-1-the?utm_campaign=post-expanded-share&utm_medium=web"
    :margin="10"
    :imageOptions="{ margin: 10 }"
/>

<!--
* Based on a blog series
  * Conclusions come out tomorrow
  * Can start with part 1 here (QR code)
-->

---

# How I got here

* Working on Ocuroot, a CI tool that maintained state in Git.

Why?

* Enabling GitOps with automations as a default
* Git repos are cheap and easy for most teams to set up
* I *really* didn't want to write a database schema

---

# Why is this relevant to Go?

* I wrote it in Go
* Used the go-git library

Will discuss Go features that helped and hindered along the way:

* Concurrency
* Profiling
* Garbage collection

---

# Git is storage

<!--
When it comes down to it, Git is just a way of storing files.
It does a bunch of fancy things, like versioning, branching, diffs.
But at the end of the day, you're just putting files onto a shared server.
-->

---

# What else is file storage?

<!--
Object storage, buckets, S3. Those are all the same thing, and I'm not sure which one I prefer.
I'm just going to say S3, even though all my testing was done on Cloudflare's R2.
-->

---

# What could you use git for instead of S3?

* A REST API to store and retrieve one file at a time

<!--
Avoids transactions
Simple to understand and work with.
-->

---

# What is REST?

<!--
Some of you may know what REST is, some of you may not.
But I've got 30 minutes and this slide made the cut.
-->

---

# HTTP Verbs

* GET
* POST
* PUT
* DELETE

<!--
Ignoring PATCH and OPTIONS.
-->

---

# Example

### Create a new user profile

```
POST /users/alice/profile
Content-Type: application/json
{”name”: “Alice Smith”, “email”: “alice@example.com”, “role”: “developer”}
→ 201 Created
```

### Retrieve the profile

``` 
GET /users/alice/profile
→ 200 OK
→ {”name”: “Alice Smith”, “email”: “alice@example.com”, “role”: “developer”}
```

---

# Example

### Update the profile 
```
PUT /users/alice/profile
Content-Type: application/json
{”name”: “Alice Smith”, “email”: “alice@newdomain.com”, “role”: “senior developer”}
→ 204 No Content
```

### Delete the profile

```
DELETE /users/alice/profile
→ 204 No Content
```

---

# The interface

```go
type APIBackend interface {
	GET(ctx context.Context, path string) ([]byte, error)
	POST(ctx context.Context, path string, body []byte) error
	PUT(ctx context.Context, path string, body []byte) error
	DELETE(ctx context.Context, path string) error
}
```

Compare different implementations:

* Memory
* Git
* S3

<!--
Implement this so we can swap out with multiple backends
-->

---
layout: cover
background: images/part1/cover.jpg
---

# First attempt with Git

A naïve implementation

---
layout: quote
---

# Using the Git CLI

```go  
  cmd = exec.Command("git", "commit", "-m", message)
  cmd.Dir = repoPath
  if err := cmd.Run(); err != nil {
      return fmt.Errorf("git commit: %w", err)
  }
  
  cmd = exec.Command("git", "push")
  cmd.Dir = repoPath
  if err := cmd.Run(); err != nil {
      return fmt.Errorf("git push: %w", err)
  }
```

<!--
Just call git using exec
Clone the repo to a temp directory, read files directly for get, create and push commits for changes.
-->

---
layout: quote
---

# Keeping up to date

```go
  cmd := exec.Command("git", "pull")
  cmd.Dir = repoPath
  if err := cmd.Run(); err != nil {
      return fmt.Errorf("git pull: %w", err)
  }
  
  filePath := filepath.Join(repoPath, filename)
  return os.ReadFile(filePath)
```

<!--
Need to pull before every read to make sure we're in sync with the remote.
-->

---

# Testing

<div class="flex items-center justify-center">

```mermaid {scale: 0.6}
flowchart TD
    B[GET test.json] -->|404| C[POST test.json]
    C -->|201| D[PUT test.json]
    D -->|204| E[DELETE test.json]
    E -->|204| F[Pass]
    
    B -->|Else| G[Fail]
    C -->|Else| G
    D -->|Else| G
    E -->|Else| G
    
    style F fill:#90EE90
    style G fill:#FFB6C1
```

</div>

```bash
$ go test ./backends/gitporcelain
```

<!--
I set up a test, which felt a bit slow.
I wanted to get a sense of what parts took the longest, so I dusted off the profiler.
-->

---

FELT SLUGGISH

TODO: Add an image

---
layout: quote
---

# Profiling

## Tasks and regions

```go
ctx, task = trace.NewTask(ctx, "TestGET")
defer task.End()

...

defer trace.StartRegion(ctx, "GET").End()
```

## Create and view a trace

```bash
go test ./backends/gitporcelain -trace=trace.out 
go tool trace trace.out
```

---
layout: center
class: text-center
---

![Trace output](/images/part1/trace_porcelain.png)

---
layout: center
---

# Compare to S3

---
layout: center
class: text-center
---

![Trace output](/images/part1/trace_s3.png)

---
layout: cover
background: images/part2/cover.jpg
---

# Something faster

Down to the protocol level

---

# The layers of git

<div class="flex justify-center">
<svg width="700" height="350" xmlns="http://www.w3.org/2000/svg">
  <!-- Porcelain Layer -->
  <rect x="50" y="30" width="600" height="90" fill="#E74C3C" stroke="#333" stroke-width="2"/>
  <text x="350" y="85" text-anchor="middle" fill="white" font-size="12" font-weight="bold">Porcelain Commands</text>
  
  <!-- Plumbing Layer -->
  <rect x="50" y="120" width="600" height="90" fill="#16A085" stroke="#333" stroke-width="2"/>
  <text x="350" y="175" text-anchor="middle" fill="white" font-size="12" font-weight="bold">Plumbing Commands</text>
  
  <!-- Protocol Layer -->
  <rect x="50" y="210" width="295" height="90" fill="#2980B9" stroke="#333" stroke-width="2"/>
  <text x="197" y="265" text-anchor="middle" fill="white" font-size="12" font-weight="bold">Protocols</text>
  
  <!-- .git Directory -->
  <rect x="345" y="210" width="305" height="90" fill="#27AE60" stroke="#333" stroke-width="2"/>
  <text x="497" y="265" text-anchor="middle" fill="white" font-size="12" font-weight="bold">.git</text>
</svg>
</div>

---

# Focusing on protocols

<div class="flex justify-center">
<svg width="700" height="350" xmlns="http://www.w3.org/2000/svg">
  <!-- Porcelain Layer -->
  <rect x="50" y="30" width="600" height="90" fill="#E74C3C" stroke="#333" stroke-width="2"/>
  <text x="350" y="85" text-anchor="middle" fill="white" font-size="12" font-weight="bold">Porcelain Commands</text>
  
  <!-- Plumbing Layer -->
  <rect x="50" y="120" width="600" height="90" fill="#16A085" stroke="#333" stroke-width="2"/>
  <text x="350" y="175" text-anchor="middle" fill="white" font-size="12" font-weight="bold">Plumbing Commands</text>
  
  <!-- Protocol Layer -->
  <rect x="50" y="210" width="295" height="90" fill="#2980B9" stroke="#333" stroke-width="2"/>
  <text x="197" y="265" text-anchor="middle" fill="white" font-size="12" font-weight="bold">Protocols</text>
  
  <!-- .git Directory -->
  <rect x="345" y="210" width="305" height="90" fill="#27AE60" stroke="#333" stroke-width="2"/>
  <text x="497" y="265" text-anchor="middle" fill="white" font-size="12" font-weight="bold">.git</text>
  
  <!-- Highlight circle around protocols -->
  <ellipse cx="197" cy="255" rx="140" ry="45" fill="none" stroke="#FF0000" stroke-width="4"/>
</svg>
</div>

---

# Introducing go-git

TODO: Screenshot of the go-git project page

---

# Connecting

```go
// Set up a transport for this endpoint
ep, err := transport.NewEndpoint("https://github.com/theothertomelliott-test/test-repo.git")
t, err := transport.Get(ep.Scheme)

// Establish a session
sess, err := t.NewSession(memory.NewStorage(),
    ep,
    http.BasicAuth{
        Username: “git”,
        Password: githubToken,
    },
)

// Separate handshakes are needed for reading and writing
readConn, err := sess.Handshake(ctx, transport.UploadPackService, "")
writeConn, err := sess.Handshake(ctx, transport.ReceivePackService, "")
```

---

# Keeping up to date

```go
refs, err := conn.GetRemoteRefs(ctx)

for _, ref := range refs {
    if ref.Name().IsBranch() && ref.Name().String() == branch {
        return ref.Hash()
    }
}
```

---

# Trees?

```
.
├── dir1
│   └── dir2
│       └── hello2.txt
└── hello.txt
```

---

# Trees!

It's all objects

![Tree](/images/part2/tree.png)

---
layout: image-right
image: /images/part2/onlywhatyouneed.png
---

# Fetch only what you need!

```go
err := conn.Fetch(
  ctx, 
  &transport.FetchRequest{
    Wants: []plumbing.Hash{hash},
    Filter: packp.FilterBlobLimit(
      0, // No blobs, only trees 
      packp.BlobLimitPrefixNone,
    ),
  },
)
```

---

# Building a commit

TODO: Constructing a commit object, but only including the relevant pieces

Zipping into a packfile

---
layout: cover
---

# Is it faster?

---

![Speed](/images/part2/trace.png)

---
layout: image-right
image: /images/part2/success.jpg
---

# It is faster!

<div class="flex flex-col items-center justify-center space-y-12 h-96">
  <div class="text-center">
    <div class="text-6xl font-bold text-blue-600">942ms</div>
    <div class="text-xl mt-2">Git Protocols</div>
  </div>
  <div class="text-center">
    <div class="text-6xl font-bold text-orange-600">1.2s</div>
    <div class="text-xl mt-2">S3</div>
  </div>
</div>

---
layout: cover
background: images/part3/cover.jpg
---

# Concurrency

Handling overlapping requests

---

# Writes aren't atomic

<div class="flex items-center justify-center h-96">

| Time | Push 1 | Push 2 |
|------|--------|--------|
| t=1 | <span class="inline-block bg-green-500 rounded text-white px-3 py-1">git add</span> | - |
| t=2 | <span class="inline-block bg-blue-500 rounded text-white px-3 py-1">git commit</span> | <span class="inline-block bg-green-500 rounded text-white px-3 py-1">git add</span> |
| t=3 | <span class="inline-block bg-purple-500 rounded text-white px-3 py-1">git push</span> | <span class="inline-block bg-blue-500 rounded text-white px-3 py-1">git commit</span> |
| t=4 | - | <span class="inline-block bg-purple-500 rounded text-white px-3 py-1">git push</span> |

</div>

---

# Backoff & retry

```go
import "github.com/cenkalti/backoff/v5"

...

operation := func() (plumbing.Hash, error) {
    commit, err := b.doPOST(ctx, path, body)
    if err != nil {
        if err == gitbackedrest.ErrConflict {
            return plumbing.ZeroHash, backoff.Permanent(gitbackedrest.ErrConflict)
        }
        return plumbing.ZeroHash, err
    }
    return commit, nil
}

_, err := backoff.Retry(
    ctx, 
    operation, 
    backoff.WithBackOff(backoff.NewExponentialBackOff()),
)
```

---

# This can be slow

---

# Locking

```go
func (b *Backend) POST(
    ctx context.Context, 
    path string, 
    body []byte,
) error {

    b.writeMtx.Lock()
    defer b.writeMtx.Unlock()

    ...
}
```

---

# But why both?

---
layout: cover
---

# Productionizing

Making it stable

---

TODO: flesh out this section

https://thefridaydeploy.substack.com/p/can-git-back-a-rest-api-part-4-stability

---
layout: cover
---

# Conclusions

Why would you do this?

---

TODO: flesh out this section

---
layout: cover
---

# What's next?

---

* Git-backed NFS?
* Maybe that's taking it too far.