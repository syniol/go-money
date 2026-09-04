# Security Policy

## Supported Versions

Only the latest minor release line receives security fixes. Earlier releases
must be upgraded before a fix can be back-ported.

| Version | Supported |
|---------|-----------|
| latest v1.x | yes |
| earlier v1.x | no |
| v0.x | no |

## Reporting a Vulnerability

Please report suspected vulnerabilities privately using GitHub's
"Report a vulnerability" workflow on the repository's Security tab:

https://github.com/syniol/go-money/security/advisories/new

Do not open a public issue for a vulnerability report. Public issues let
attackers weaponise the fix window.

When reporting, please include:

- A short description of the issue class (e.g. "integer overflow in parser").
- A minimal reproducer, ideally a Go test that fails on the current tip.
- The Git SHA or release tag you tested against.
- The impact you believe the issue has (data corruption, denial of service, panic on trusted input, etc.).

You will receive an acknowledgement within five working days. A fix
timeline is agreed with you before disclosure, typically within 30 days
for high-severity issues.

## Scope

In scope:

- Integer overflow or underflow in any exported arithmetic function.
- Panics on adversarially-crafted input to `NewFromString`, `UnmarshalJSON`, `UnmarshalText` or `FromDecimal`.
- Incorrect rounding, currency mixing, or precision loss that could enable value manipulation.
- Any behaviour that contradicts the compatibility contract stated in `README.md` or `CHANGELOG.md`.

Out of scope:

- Bugs in `golang.org/x/text` transitively depended upon; report those upstream and open a tracking issue here.
- Bugs in code that constructs its own `*Currency` values outside `currencyConfig`. External construction is unsupported.
- CI misconfiguration in this repository's GitHub Actions workflows.

## Disclosure

Once a fix ships and users have had a reasonable window to upgrade, the
advisory is made public. We do not embargo indefinitely.
