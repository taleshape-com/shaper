# Security Policy

The Shaper team takes the security of our software and users seriously. We appreciate your efforts to responsibly disclose vulnerabilities.

## Supported Versions

Only the latest release and the current `main` branch receive security updates.

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < latest| :x:                |

We strongly encourage all users and self-hosters to keep their installations updated to the latest release.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, pull requests, or public discussions.**

### Preferred Method: GitHub Private Vulnerability Reporting

The fastest and most secure way to report a vulnerability is via GitHub:

1. Navigate to the [Security Advisories](https://github.com/taleshape-com/shaper/security/advisories/new) tab of the repository.
2. Click **"Report a vulnerability"**.
3. Fill out the advisory form with as much detail as possible.

### Alternative Method: Email

If you cannot use GitHub Security Advisories, you can email us directly at **[hi@taleshape.com](mailto:hi@taleshape.com)**.

Please include:
- A clear description of the vulnerability and its potential impact.
- Step-by-step instructions or a minimal Proof of Concept (PoC) to reproduce the issue.
- The version(s) of Shaper affected.
- Any relevant logs, configuration details, or proposed fixes.
- If applicable, whether the issue has been shared with any third party.

## What to Expect

When you submit a report:

1. **Acknowledgment**: We aim to acknowledge receipt of your report within 48 hours.
2. **Assessment**: We will investigate and confirm whether the issue is a vulnerability, determine its severity, and keep you informed of our progress.
3. **Resolution**: Once verified, we will work on a fix and prepare a patched release.
4. **Coordinated Disclosure**: We follow responsible disclosure practices. We will coordinate the release date and publish a security advisory giving appropriate credit to the reporter (unless you prefer to remain anonymous).

## Security Best Practices for Deployments

When running Shaper in production:
- Keep your deployment updated with the latest Docker images or binary releases.
- Protect your environment variables and sensitive configuration files.
- Ensure database credentials, JWT secret keys, and other secrets are generated securely and kept confidential.
- Use TLS/HTTPS in front of Shaper when serving over a public network.

## Verifying Release Signatures and Provenance

Shaper uses cryptographically verifiable signatures and build provenance (via Sigstore and GitHub Artifact Attestations) across release channels.

### Docker Hub Images
Shaper container images are signed keylessly with Cosign and attested with GitHub Artifact Attestations. To verify an image:

```sh
cosign verify \
  --certificate-identity-regexp "^https://github.com/taleshape-com/shaper/" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  taleshape/shaper:<tag>
```

Or using GitHub CLI:
```sh
gh attestation verify oci://taleshape/shaper:<tag> --owner taleshape-com
```

### GitHub Releases (Binaries)
Release binaries and `SHA256SUMS` include a detached Sigstore Cosign signature bundle (`SHA256SUMS.cosign.bundle`) and GitHub Artifact Attestations.

1. **Verify `SHA256SUMS` with Cosign:**
   ```sh
   cosign verify-blob \
     --bundle SHA256SUMS.cosign.bundle \
     --certificate-identity-regexp "^https://github.com/taleshape-com/shaper/" \
     --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
     SHA256SUMS
   ```

2. **Verify downloaded binary against checksums:**
   ```sh
   sha256sum -c SHA256SUMS --ignore-missing
   ```

3. **Or verify using GitHub CLI:**
   ```sh
   gh attestation verify <binary-file> --owner taleshape-com
   ```

### npm & PyPI
- **npm**: Published using `--provenance` via OIDC Trusted Publishing, verifiable on [npmjs.com](https://www.npmjs.com/package/@taleshape/shaper) or via `npm audit signatures`.
- **PyPI**: Published via OIDC Trusted Publishing with PEP 740 digital attestations and GitHub provenance attestations.

