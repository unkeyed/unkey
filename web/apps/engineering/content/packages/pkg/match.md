---
title: match
description: "provides pattern matching utilities"
---

Package match provides pattern matching utilities.

## Functions

### func Wildcard

```go
func Wildcard(s, pattern string) (bool, error)
```

Wildcard checks if a string matches a wildcard pattern. The pattern can contain '\*' as a wildcard that matches any sequence of characters.

Examples:

  - Wildcard("test@gmail.com", "\*@gmail.com") returns true
  - Wildcard("test@yahoo.com", "\*@gmail.com") returns false
  - Wildcard("hello world", "hello\*") returns true
  - Wildcard("hello world", "\*world") returns true
  - Wildcard("hello world", "h\*d") returns true

