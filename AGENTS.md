# Agent Guidelines for Functionless URL Shortener

Welcome to the workspace-specific agent guidelines. These instructions apply to AI agents and developers working on the `functionless-url-shortener` codebase.

---

## 1. Project Architecture & Layout

This project implements a **100% functionless architecture** on AWS:

- `template.yaml`: AWS Serverless Application Model (SAM) CloudFormation template. Defines:
  - `SiteAPI`: Regional API Gateway REST API with IAM/SigV4 authorization and resource policies. Embeds `api.yaml` via `Fn::Transform: AWS::Include`.
  - `LinkTable`: Amazon DynamoDB table with PAY_PER_REQUEST billing, `id` (hash key), `owner` (GSI), and TTL (`ttl` attribute).
  - `CloudFrontDistro`: CloudFront CDN distribution caching public redirects (`/?go=...`) and accelerating global traffic.
  - `DDBReadRole` & `DDBCrudRole`: IAM execution roles assumed by `apigateway.amazonaws.com` for DynamoDB `GetItem` and `UpdateItem`.
  - CloudWatch Alarms & SNS Topic: Operational monitors for API 4xx/5xx/Latency, DynamoDB 4xx/5xx, and CloudFront error/cache hit rates.
- `api.yaml`: OpenAPI 3.0.1 specification detailing:
  - `/` & `/{linkId}`: Public HTTP 301 redirects using VTL mapping templates directly into DynamoDB `GetItem`.
  - `/api` (`POST`): Secured link creation using DynamoDB `UpdateItem` with conditional check `attribute_not_exists(id)` and SigV4 IAM auth.
  - `/api/{linkId}` (`GET`): Secured link metadata retrieval using DynamoDB `GetItem` and SigV4 IAM auth.

---

## 2. Core Engineering Principles & Safety

### Functionless Direct Service Integration
- **Zero Compute:** No AWS Lambda functions or custom containers are used for request handling.
- **Velocity Template Language (VTL):** All data mapping, status code overrides, and conditional responses are handled in API Gateway VTL request/response mapping templates.

### Cloud Mutating Safety & Dry-Run
- When deploying CloudFormation or SAM changes, always use change sets or guided validation:
  ```bash
  sam deploy --no-execute-changeset
  ```
- Do not make destructive schema changes to DynamoDB table keys or API Gateway resource policies without verifying backwards compatibility.

---

## 3. Conventional Commits & Versioning

This project strictly follows the [Conventional Commits](https://www.conventionalcommits.org/) specification with automated releases managed via Release Please.
- `feat:` minor version bump
- `fix:` patch version bump
- `docs:`, `chore:`, `refactor:`, `test:`
- `feat!:` or `fix!:` (or `BREAKING CHANGE:` footer) major version bump

---

## 4. Living Product Roadmap Management

`ROADMAP.md` is the central living document tracking future capabilities, optimizations, and technical debt:
- **Adding Items**: Whenever you or the user identify a capability, optimization, or edge-case improvement for future work, add it to `ROADMAP.md` under the appropriate category.
- **Deduplication with GitHub Issues**: If an active GitHub Issue already exists or is explicitly created for a feature, bug fix, or task, **do not duplicate it in `ROADMAP.md`**. GitHub Issues track active, assigned, or triaged tasks, while `ROADMAP.md` captures high-level, unassigned architectural vision and backlog capabilities.
- **Removing Items**: Once a feature is fully implemented, verified with tests, and committed, **remove it from `ROADMAP.md`** immediately to keep the roadmap focused on active upcoming tasks.

---

## 5. Verification Commands

Before submitting any changes, verify template integrity:
```bash
make validate
```
Or directly:
```bash
sam validate -t template.yaml
```

