package templates

// Contributing returns a CONTRIBUTING.md template.
func Contributing(repoName string) string {
	return `# Contributing to ` + repoName + `

Thank you for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch (` + "`git checkout -b feature/my-feature`" + `)
3. Commit your changes (` + "`git commit -am 'Add my feature'`" + `)
4. Push to the branch (` + "`git push origin feature/my-feature`" + `)
5. Open a Pull Request

## Code Style

Please follow the existing code style and conventions.

## Reporting Bugs

Open an issue with a clear description and steps to reproduce.

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.
`
}

// CodeOfConduct returns a Contributor Covenant Code of Conduct.
func CodeOfConduct() string {
	return `# Contributor Covenant Code of Conduct

## Our Pledge

We as members, contributors, and leaders pledge to make participation in our
community a harassment-free experience for everyone, regardless of age, body
size, visible or invisible disability, ethnicity, sex characteristics, gender
identity and expression, level of experience, education, socio-economic status,
nationality, personal appearance, race, religion, or sexual identity
and orientation.

## Our Standards

Examples of behavior that contributes to a positive environment:

* Using welcoming and inclusive language
* Being respectful of differing viewpoints and experiences
* Gracefully accepting constructive criticism
* Focusing on what is best for the community
* Showing empathy towards other community members

Examples of unacceptable behavior:

* The use of sexualized language or imagery and unwelcome sexual attention
* Trolling, insulting or derogatory comments, and personal or political attacks
* Public or private harassment
* Publishing others' private information without explicit permission
* Other conduct which could reasonably be considered inappropriate

## Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be
reported to the project maintainers. All complaints will be reviewed and
investigated promptly and fairly.

## Attribution

This Code of Conduct is adapted from the [Contributor Covenant](https://www.contributor-covenant.org),
version 2.1.
`
}

// SecurityPolicy returns a SECURITY.md template.
func SecurityPolicy(repoName string) string {
	return `# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in ` + repoName + `, please report it responsibly.

**Do not open a public issue.**

Instead, please send an email or use GitHub's private vulnerability reporting feature.

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | Yes       |

## Response Timeline

We aim to acknowledge security reports within 48 hours and provide a fix within 7 days for critical issues.
`
}
