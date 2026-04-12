# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in FUSE, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email security concerns to the maintainers via the [Uranus Technologies](https://uranus.com.br) contact page or open a private security advisory on GitHub:

1. Go to the [Security Advisories](https://github.com/open-source-cloud/fuse/security/advisories) page
2. Click "New draft security advisory"
3. Provide a description of the vulnerability and steps to reproduce

We will acknowledge receipt within 48 hours and provide a timeline for a fix.

## Supported Versions

| Version | Supported |
| ------- | --------- |
| Latest beta | Yes |

## Security Measures

- Container images are scanned with [Trivy](https://github.com/aquasecurity/trivy) on every release
- Production images use distroless base (gcr.io/distroless/static-debian12)
- Containers run as non-root user
- Static binary with CGO disabled
- Code quality enforced via golangci-lint v2
