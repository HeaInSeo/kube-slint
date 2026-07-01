# Third-Party Licenses

This project is licensed under Apache-2.0. This file summarizes third-party Go modules used by kube-slint as of 2026-06-27.

Source of dependency inventory:

```sh
go list -m all
```

## Direct Dependencies

| Module | Version | License |
|---|---:|---|
| `github.com/onsi/ginkgo/v2` | v2.22.0 | MIT |
| `github.com/stretchr/testify` | v1.11.1 | MIT |
| `gopkg.in/yaml.v3` | v3.0.1 | Apache-2.0 / MIT notice in upstream package |

## Material Indirect Dependencies

| Module | Version | License |
|---|---:|---|
| `github.com/onsi/gomega` | v1.36.1 | MIT |
| `golang.org/x/crypto` | v0.46.0 | BSD-3-Clause |
| `golang.org/x/mod` | v0.30.0 | BSD-3-Clause |
| `golang.org/x/net` | v0.48.0 | BSD-3-Clause |
| `golang.org/x/sync` | v0.18.0 | BSD-3-Clause |
| `golang.org/x/sys` | v0.39.0 | BSD-3-Clause |
| `golang.org/x/term` | v0.38.0 | BSD-3-Clause |
| `golang.org/x/text` | v0.32.0 | BSD-3-Clause |
| `golang.org/x/tools` | v0.39.0 | BSD-3-Clause |
| `google.golang.org/protobuf` | v1.36.11 | BSD-3-Clause |

## Additional Transitive Go Modules

These modules are pulled transitively by tests, development tooling, or the packages above:

| Module | Version |
|---|---:|
| `github.com/chzyer/readline` | v1.5.1 |
| `github.com/creack/pty` | v1.1.9 |
| `github.com/davecgh/go-spew` | v1.1.1 |
| `github.com/go-logr/logr` | v1.4.2 |
| `github.com/go-task/slim-sprig/v3` | v3.0.0 |
| `github.com/golang/protobuf` | v1.5.0 |
| `github.com/google/go-cmp` | v0.7.0 |
| `github.com/google/pprof` | v0.0.0-20241029153458-d1b30febd7db |
| `github.com/ianlancetaylor/demangle` | v0.0.0-20240312041847-bd984b5ce465 |
| `github.com/kr/pretty` | v0.3.1 |
| `github.com/kr/pty` | v1.1.1 |
| `github.com/kr/text` | v0.2.0 |
| `github.com/pkg/diff` | v0.0.0-20210226163009-20ebb0f2a09e |
| `github.com/pmezard/go-difflib` | v1.0.0 |
| `github.com/rogpeppe/go-internal` | v1.13.1 |
| `github.com/stretchr/objx` | v0.5.2 |
| `github.com/yuin/goldmark` | v1.4.13 |
| `golang.org/x/telemetry` | v0.0.0-20251111182119-bc8e575c7b54 |
| `gopkg.in/check.v1` | v1.0.0-20201130134442-10cb98267c6c |

The authoritative license terms are the upstream license files shipped with each module.
