# Security Policy

## Reporting a Vulnerability

We take security seriously. We appreciate your efforts to responsibly disclose your findings.

### How to Report a Security Issue

**Please DO NOT report security vulnerabilities through public GitHub issues.**

Instead, please report security vulnerabilities by emailing:

📧 **support@nannyai.dev**

We will acknowledge your report within 48 hours and provide an estimated timeframe for a fix.

## System Prompts & AI Safety

To prevent prompt injection attacks and ensure the reliability of our AI agents, the core system prompts are **not public**.

- **Self-Hosted Access**: If you are running a self-hosted instance and require access to the system prompts, please contact support with your use case.
- **Verification**: We are actively working on verifying our prompts against adversarial attacks. Once we are confident in their robustness, we plan to make them public.

## Patch Integrity

All patch scripts distributed by NannyAPI are verified using SHA256 hashes.
- The API serves the script content along with its expected hash.
- The agent calculates the hash of the downloaded script.
- Execution proceeds **only** if the hashes match.

This mechanism protects against Man-in-the-Middle (MITM) attacks during script delivery.
