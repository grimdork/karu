# Kāru

A Go library for forum and comment system logic, agnostic of the consuming application.

## Features

- **Dual database:** PostgreSQL via pgx and SQLite via modernc.org/sqlite
- **Hierarchical paths:** Organise content under arbitrary paths (e.g. `forum/general`, `blog/post-123`, `gallery/image-456`)
- **Flexible permissions:** Permission groups with codes (`r` read, `w` post, `d` delete own, `D` delete any, `t` delete thread, `l` lock/unlock, `m` move)
- **Path-inherited permissions:** Strictest permission across all matching path levels applies
- **Silent denial:** Paths and posts without access are invisible
- **Threaded replies:** Full nesting of posts via `parent_id`
- **User preferences:** Per-user key-value storage for filtering and settings
- **Search:** Title and content search across posts
- **No account system:** The consuming application handles authentication and passes user IDs and permission strings with each API call

## Usage

```go
import "github.com/grimdork/karu"

k, err := karu.New("sqlite", "file::memory:?cache=shared")
if err != nil { /* handle */ }
defer k.Close()

ctx := context.Background()
if err := k.Migrate(ctx); err != nil { /* handle */ }

post, err := k.CreateThread(ctx, "general", "user1",
    "Hello World", "First post!", "general:rwd")
if err != nil { /* handle */ }

reply, err := k.CreatePost(ctx, post.ID, "user2",
    "Nice reply", "general:rwd")
if err != nil { /* handle */ }
```

## Permission strings

Format: `keyword:code[,keyword:code]*`

Example: `general:rwdDtlm,moderators:rw`

Permission codes:
| Code | Meaning |
|------|---------|
| `r`  | Read posts |
| `w`  | Create posts/replies |
| `d`  | Delete own posts |
| `D`  | Delete any post |
| `t`  | Delete threads |
| `l`  | Lock/unlock threads |
| `m`  | Move threads/posts |

Permission evaluation walks from the most specific path to root and takes the intersection of all matching groups.
