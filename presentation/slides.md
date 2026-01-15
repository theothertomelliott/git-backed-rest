---
# try also 'default' to start simple
theme: default
background: /images/intro/cover.jpg
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
# force color schema for the slides, can be 'auto', 'light', or 'dark'
colorSchema: dark
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

---
layout: image-right
image: images/intro/profile.png
---

# About me

Tom Elliott

* 20 year developer
* 10 year Gopher
* Failed founder (no regrets)

<!--
Hi everyone, I'm Tom, a 20 year developer, 10 year Gopher, and a failed founder.

The last month or two, among some interviews, I've been doing some fun research projects, one of which I'm going to talk about today.
-->

---

# What we're talking about
Expanding on a blog series


<img src="/images/intro/blog.png" alt="Screenshot of the first in the series" style="max-width: 70%; margin: 0 auto; display: block;">

<!--
* Based on a blog series
  * Conclusions come out tomorrow
  * Can start with part 1 here (QR code)
-->

---
layout: cover
---

# Why would I do this?

---
layout: image
image: images/intro/ocuroot.png
---

<!--
# Why would I do all this?

* Working on Ocuroot, a CI tool that maintained state in Git.

* Enabling GitOps with automations as a default
* Git repos are cheap and easy for most teams to set up
* I *really* didn't want to write a database schema
-->

---
layout: two-cols-header
---

# What I'll cover

<Spacer size="2rem" />

::left::

## Git Stuff

<Spacer size="1rem" />

* Protocols
* Objects & Tree structure
* Packfiles

::right::

## Go Stuff

<Spacer size="1rem" />

* Modules I used
* Profiling
* Concurrency
* Garbage collection

---

# What is REST?

https://www.geeksforgeeks.org/node-js/rest-api-introduction/

![Screenshot of REST API introduction](/images/intro/rest-api.png)

<!--
Some of you may know what REST is, some of you may not.
But I've got 30 minutes and this slide made the cut.
-->

---

# HTTP Verbs

<Spacer size="1rem" />

| Method | Description |
|--------|-------------|
| <span style="display: inline-block; background-color: #4caf50; color: white; padding: 0.3rem 0.8rem; border-radius: 1rem; font-weight: 600;">POST</span> | Create |
| <span style="display: inline-block; background-color: #61dafb; color: #000; padding: 0.3rem 0.8rem; border-radius: 1rem; font-weight: 600;">GET</span> | Read |
| <span style="display: inline-block; background-color: #ff9800; color: white; padding: 0.3rem 0.8rem; border-radius: 1rem; font-weight: 600;">PUT</span> | Update |
| <span style="display: inline-block; background-color: #f44336; color: white; padding: 0.3rem 0.8rem; border-radius: 1rem; font-weight: 600;">DELETE</span> | Delete |

<!--
Ignoring PATCH and OPTIONS.
-->

---

# Example

<Spacer size="2rem" />

## Create a new user profile

```
POST /users/alice/profile
Content-Type: application/json
{”name”: “Alice Smith”, “email”: “alice@example.com”, “role”: “developer”}
→ 201 Created
```

<Spacer size="2rem" />

## Retrieve the profile

``` 
GET /users/alice/profile
→ 200 OK
→ {”name”: “Alice Smith”, “email”: “alice@example.com”, “role”: “developer”}
```

---

# Example

<Spacer size="2rem" />

## Update the profile 
```
PUT /users/alice/profile
Content-Type: application/json
{”name”: “Alice Smith”, “email”: “alice@newdomain.com”, “role”: “senior developer”}
→ 204 No Content
```

<Spacer size="2rem" />

## Delete the profile

```
DELETE /users/alice/profile
→ 204 No Content
```

---

# The interface

<Spacer size="3rem" />

```go
type APIBackend interface {
	GET(ctx context.Context, path string) ([]byte, error)
	POST(ctx context.Context, path string, body []byte) error
	PUT(ctx context.Context, path string, body []byte) error
	DELETE(ctx context.Context, path string) error
}
```

<!--
Implement this so we can swap out with multiple backends

One backend for Git, one for S3
-->

---
layout: cover
background: images/part1/cover.jpg
---

# First attempt with Git

A naïve implementation

<!--
Photo by BOOM 💥 Photography: https://www.pexels.com/photo/person-starting-on-running-block-12585946/
-->

---

# Using the Git CLI

<Spacer size="2rem" />

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

# Keeping up to date

<Spacer size="2rem" />
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

# Test setup
On-demand repos in a test GitHub org

![Test GitHub Organization](/images/part1/test-org.png)

---

# Test sequence

<div class="flex items-center justify-center text-sm mb-8">

| Method | Path | Request Body | Code | Response Body |
|--------|------|-------------|------|--------------|
| <span class="inline-block bg-blue-500 rounded text-white px-2 py-1 text-xs">GET</span> | test.json | - | 404 | - |
| <span class="inline-block bg-green-500 rounded text-white px-2 py-1 text-xs">POST</span> | test.json | `{"num": 1}` | 201 | - |
| <span class="inline-block bg-blue-500 rounded text-white px-2 py-1 text-xs">GET</span> | test.json | - | 200 | `{"num": 1}` |
| <span class="inline-block bg-green-500 rounded text-white px-2 py-1 text-xs">POST</span> | test.json | `{"num": 2}` | 409 | - |

</div>

```bash
$ go test ./backends/gitporcelain
```

<!--
I set up a test, which felt a bit slow.
I wanted to get a sense of what parts took the longest, so I dusted off the profiler.
-->

---
layout: image
image: images/part1/slug.jpg
---

<!--
Photo by Pixabay: https://www.pexels.com/photo/slug-158158/
-->

---

# Profiling

<Spacer size="2rem" />

## Tasks and regions

```go
ctx, task = trace.NewTask(ctx, "TestGET")
defer task.End()

...

defer trace.StartRegion(ctx, "GET").End()
```

<Spacer size="2rem" />

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
layout: cover
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
https://github.com/go-git/go-git

<Spacer size="1rem" />

![go-git readme](/images/part2/go-git-readme.png)

---

# Connecting

<Spacer size="1rem" />

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

<Spacer size="2rem" />

```go
refs, err := conn.GetRemoteRefs(ctx)

for _, ref := range refs {
    if ref.Name().IsBranch() && ref.Name().String() == branch {
        return ref.Hash()
    }
}
```

---

# What's in a commit?

<Spacer size="2rem" />

```
author Tom Elliott <...> 1766439004 -0500
committer Tom Elliott <...> 1766439004 -0500

Add all files
```

<Spacer size="2rem" />

<pre style="font-size: 1.2em; font-family: monospace; line-height: 1.4; background-color: #2d2d2d; color: #f8f8f2; padding: 1rem; border-radius: 4px;">
.
├── dir1
│   └── dir2
│       └── hello2.txt
└── hello.txt
</pre>

---

# Objects!

Commits, trees, blobs (and tags)

<img src="/images/part2/tree.svg" alt="Tree" style="max-width: 80%; height: 80%; margin: 0 auto; display: block;">

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

# Packfiles
Zip up changes

<img src="/images/part2/packfile.svg" alt="Packfile" style="max-width: 80%; height: 80%; margin: 0 auto; display: block;">


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
layout: image
image: images/part1/slug.jpg
---

<!--
Photo by Pixabay: https://www.pexels.com/photo/slug-158158/
-->

---

# Locking

<Spacer size="1rem" />

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

<Spacer size="1rem" />

<img src="/images/part3/both.jpg" alt="Two drinks" style="max-width: 90%; height: auto; margin: 0 auto; display: block;">

<!--
Photo by damla selen demir: https://www.pexels.com/photo/coffee-and-drink-26987449/
-->

---
layout: cover
background: images/part4/cover.jpg
---

# Productionizing

Making it stable

---
layout: image
image: images/part4/grafana.png
backgroundSize: contain
---

---

# Test Sequence

<div class="flex items-center justify-center text-sm mb-8">

| Operation | Method | Path | Request Body | Expected Code |
|-----------|--------|------|-------------|--------------|
| **Create** | <span class="inline-block bg-green-500 rounded text-white px-2 py-1 text-xs">POST</span> | `{random1}.json` | *random data* | 201 |
| **Create** | <span class="inline-block bg-green-500 rounded text-white px-2 py-1 text-xs">POST</span> | `{random2}.json` | *random data* | 201 |
| **Read** | <span class="inline-block bg-blue-500 rounded text-white px-2 py-1 text-xs">GET</span> | `{random1}.json` | - | 200 |
| **Read** | <span class="inline-block bg-blue-500 rounded text-white px-2 py-1 text-xs">GET</span> | `{random2}.json` | - | 200 |
| **Update** | <span class="inline-block bg-orange-500 rounded text-white px-2 py-1 text-xs">PUT</span> | `{random1}.json` | *random data* | 204 |
| **Delete** | <span class="inline-block bg-red-500 rounded text-white px-2 py-1 text-xs">DELETE</span> | `{random2}.json` | - | 204 |

</div>

---

# Configurable tests

```bash
$ go run ./cmd/uptime_test -repetitions=4 -duration=2h -filesize=1024
```

<div class="flex items-center justify-center text-sm mb-8">

| Parameter | Flag | Description | Example |
|-----------|------|-------------|---------|
| **Repetitions** | `-repetitions` | Number of test sequences per minute | `-repetitions=4` |
| **Duration** | `-duration` | Total test run time | `-duration=10m` |
| **File Size** | `-filesize` | Size of test data in kb | `-filesize=1024` |

</div>

---

# Low volume run

```bash
$ go run ./cmd/uptime_test -repetitions=1 -duration=1h -filesize=1
```

![Low Volume](/images/part4/graph1.png)

---

# Rising memory usage

Not a great sign

![Low Volume](/images/part4/graph2.png)

---

# Bigger files, slower requests

```bash
$ go run ./cmd/uptime_test -repetitions=1 -duration=1h -filesize=1024
```

![Low Volume](/images/part4/graph3.png)

---

# Actually, errors!

```
2026/01/05 16:40:16 Action #54 failed: failed to delete second resource: 
executing DELETE request: Delete "http://localhost:8080/sthenic-Mitella-batlan": 
context deadline exceeded
```

![Low Volume](/images/part4/graph4.png)

---

# Let's look at a profile

Added Pyroscope to the mix

![Low Volume](/images/part4/profile.png)

---

# Too many blobs

```go
func (b *Backend) walkTree(
  treeHash, 
  includeHash func(plumbing.Hash) error,
) error {

...

  for _, entry := range tree.Entries {
    if entry.Mode.IsFile() {
      // Object is a blob, include if it's in our store
      // HERE'S THE PROBLEM
      _, err := b.store.EncodedObject(plumbing.BlobObject, entry.Hash)
      if err == nil {
          includeHash(entry.Hash)
      }
    } else if entry.Mode == filemode.Dir {
      // It's a subtree - recurse
      b.walkTree(entry.Hash, includeHash)
    }
  }

...

}
```

---

# We only need one blob

<Spacer size="1rem" />

```go
if entry.Mode.IsFile() && entry.Hash == modifiedFileHash {
  // Object is a blob, include if it's in our store
  _, err := b.store.EncodedObject(plumbing.BlobObject, entry.Hash)
  if err == nil {
      includeHash(entry.Hash)
  }
} else if entry.Mode == filemode.Dir {
...
```

---

# Stable latency

Tracks with file size

<Spacer size="1rem" />

![Low Volume](/images/part4/graph5.png)

---

# Runaway memory usage!

No longer restricted by timeouts

<Spacer size="1rem" />

![Low Volume](/images/part4/graph6.png)

---

# Cleanup

Quick and dirty - empty the store every 10 seconds

<Spacer size="1rem" />

```go
go func() {
  // Clean up objects every 10s
  for range time.Tick(10 * time.Second) {
    b.sessionMtx.Lock()

    b.store.ObjectStorage.Objects = make(map[plumbing.Hash]plumbing.EncodedObject)
    b.store.ObjectStorage.Commits = make(map[plumbing.Hash]plumbing.EncodedObject)
    b.store.ObjectStorage.Trees = make(map[plumbing.Hash]plumbing.EncodedObject)
    b.store.ObjectStorage.Blobs = make(map[plumbing.Hash]plumbing.EncodedObject)
    b.store.ObjectStorage.Tags = make(map[plumbing.Hash]plumbing.EncodedObject)

    b.sessionMtx.Unlock()
  }
}()
```

---

# Better, but not great

50% of 2GB is still 1GB

<Spacer size="1rem" />

![Low Volume](/images/part4/graph7.png)

---

# Tuning garbage collection

```bash
GOGC=50
GOMEMLIMIT=200MiB
```

![Low Volume](/images/part4/graph8.png)

---

# The tradeoff

Increased memory usage

![Low Volume](/images/part4/graph9.png)

---

# What about S3?

It uses less memory for sure

![Low Volume](/images/part4/graph10.png)

---

# What about S3

And less CPU

![Low Volume](/images/part4/graph11.png)

<!--
I had to do a decent amount of work to get my server to this level of performance, and
S3 still blows it out of the water.

There's still some room for improvement. I could bypass the object store entirely and work more
directly with the protocol, maybe keep the objects on the stack rather than the heap.
-->

---
layout: cover
background: images/part5/cover.jpg
---

# Conclusions

Is this a good idea?

<!--
Photo credit: Photo by Martin Lopez: https://www.pexels.com/photo/man-in-black-blazer-wearing-black-framed-eyeglasses-2399065/
-->

---
layout: cover
---

# Limitations

---
layout: image
image: images/part5/hardwork.jpg
---

<!--
Photo by Mikael Blomkvist: https://www.pexels.com/photo/a-man-and-a-woman-working-at-a-construction-site-8961025/
-->

---
layout: two-cols-header
---

# File size

::left::

![Toy crane](/images/part5/toycrane.jpg)

::right::

![Large crane](/images/part5/largecrane.jpg)

<!--
Poor performance
1MB

Photo by Brett Jordan: https://www.pexels.com/photo/red-construction-crane-against-cloudy-sky-35567082/

Hard limit on GitHub
100MB
-->

---

# Rate Limits

<div class="flex items-center justify-center text-sm mb-8">

| Platform | Reads / minute | Writes / minute | Type | Source |
|----------|------------------|-------------------|------|--------|
| **GitHub (Git)** | 900 | 6 | Recommendation | [Source](https://docs.github.com/en/repositories/creating-and-managing-repositories/repository-limits#activity)
| **GitHub (REST API)** | 83 | 83 | Rate limit (standard) | [Source](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#about-primary-rate-limits)
| **GitHub (REST API)** | 250 | 250 | Rate limit (enterprise) | [Source](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#about-primary-rate-limits)
| **GitLab (SSH)** | 600 | 600 | Rate limit | [Source](https://genboree.org/gitlab/help/security/rate_limits.md#git-operations-using-ssh)
| **Google Secure Source Manager** | 1000 | 1000 | Rate limit | [Source](https://docs.cloud.google.com/secure-source-manager/docs/quotas#usage_limits)
| **AWS S3** | 300,000 | 180,000 | Throughput expectation | [Source](https://docs.aws.amazon.com/AmazonS3/latest/userguide/optimizing-performance.html)

</div>

---
layout: cover
---

# Positives

---
layout: image
image: images/part5/cheap.jpg
---
<!--
Photo by Tima Miroshnichenko: https://www.pexels.com/photo/person-holding-dollar-bills-while-using-a-calculator-6266283/
-->

---
layout: image
image: images/part5/logs.jpg
---
<!--
Photo by Aleks BM: https://www.pexels.com/photo/pile-of-brown-wooden-logs-7744517/
-->

---
layout: cover
---

# The answer!

Can Git repos replace S3 buckets?

---
layout: image
image: images/part5/drums.jpg 
---

<!--
Photo by Hernán Santarelli: https://www.pexels.com/photo/wooden-drumsticks-on-the-snare-drum-6059430/
-->

---
layout: image
image: images/part5/drums2.jpg
---

<!--
Photo by Andreu Marquès: https://www.pexels.com/photo/drumsticks-and-a-drum-set-7450047/
-->

---
layout: image
image: images/part5/drums3.jpg
---

<!--
Photo by Yan Krukau: https://www.pexels.com/photo/a-person-playing-the-drums-9010100/
-->

---

Kinda.

<!--
Go uses Git protocols to work with modules.

GitHub was built on top of a tool allowing Ruby to read/write Git:
https://deepwiki.com/mojombo/grit
https://github.com/mojombo/grit
-->

---
layout: two-cols-header
---

# Thank you!

<Spacer size="2rem" />

::left::

## Blog series

<Spacer size="1rem" />

<QRCode
    :width="300"
    :height="300"
    type="svg"
    data="https://open.substack.com/pub/thefridaydeploy/p/can-git-back-a-rest-api-part-1-the?utm_campaign=post-expanded-share&utm_medium=web"
    :margin="10"
    :imageOptions="{ margin: 10 }"
    :dotsOptions="{ color: '#000000' }"
    :backgroundOptions="{ color: '#ffffff' }"
/>

::right::

## Let's connect!

<Spacer size="1rem" />

<QRCode
    :width="300"
    :height="300"
    type="svg"
    data="https://www.linkedin.com/in/telliott1984/"
    :margin="10"
    :imageOptions="{ margin: 10 }"
    :dotsOptions="{ color: '#000000' }"
    :backgroundOptions="{ color: '#ffffff' }"
/>

---
layout: cover
---

# Questions
