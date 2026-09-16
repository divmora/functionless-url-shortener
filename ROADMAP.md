# Functionless URL Shortener Product Roadmap

This document serves as the **living product roadmap** for `functionless-url-shortener`.
- **Adding Items**: Whenever you or the user identify a capability, optimization, or edge-case improvement for future work, add it to `ROADMAP.md` under the appropriate category.
- **Deduplication with GitHub Issues**: If an active GitHub Issue already exists or is explicitly created for a feature, bug fix, or task, **do not duplicate it in `ROADMAP.md`**. GitHub Issues track active, assigned, or triaged tasks, while `ROADMAP.md` captures high-level, unassigned architectural vision and backlog capabilities.
- **Removing Items**: Once a feature is fully implemented, verified with tests, and committed, **remove it from `ROADMAP.md`** immediately to keep the roadmap focused on active upcoming tasks.

---

## 1. Routing, Redirection & Edge Optimization

- [ ] **CloudFront KeyValueStore Integration**
  - Evaluate CloudFront Functions combined with CloudFront KeyValueStore for sub-millisecond edge redirects, bypassing API Gateway for high-traffic links while maintaining regional API Gateway writes.
- [ ] **Dynamic 301 vs. 302 Redirect Mode**
  - Add support for configurable HTTP redirect status codes (`301 Moved Permanently` vs. `302 Found` / `307 Temporary Redirect`) via an optional `redirectType` attribute in DynamoDB and VTL mapping template.
- [ ] **Custom Branded Fallback / 404 Landing Page**
  - Replace the query-string redirect fallback (`error?error=url_not_found`) with a custom CloudFront origin error response serving a static HTML/CSS branded 404 landing page hosted on S3.

---

## 2. Security, Authorization & Abuse Prevention

- [ ] **Destination URL Safelisting & Malicious Domain Filtering**
  - Enforce destination domain allowlisting or integrate regex-based domain validation in the API Gateway VTL request template or JSON schema validation to prevent phishing or malicious redirects.
- [ ] **AWS WAF Integration on CloudFront**
  - Attach an AWS WAF WebACL to the CloudFront distribution with rate-limiting rules (`RateBasedRule`) and AWS Managed Rules (Common Rule Set) to mitigate brute-force URL enumeration.
- [ ] **Scoped Principal IAM Authorization**
  - Support granular IAM policy conditions allowing teams to manage only their own prefixed link slugs (e.g. `execute-api:/*/*/api/teamA-*`).

---

## 3. Observability, Analytics & Telemetry

- [ ] **Click Tracking & Telemetry Stream**
  - Enable DynamoDB Streams or CloudFront real-time log streaming to an Amazon Kinesis Firehose / S3 prefix to capture click counts, referrers, and geolocation telemetry without impacting redirect latency.
- [ ] **Amazon CloudWatch Dashboard for Link Traffic**
  - CloudFormation template definition for a CloudWatch Dashboard displaying request volume, CloudFront cache hit ratio, DynamoDB consumed read/write units, and 4xx/5xx error rates.
- [ ] **Synthetic Canary Deployment**
  - Add an AWS CloudWatch Synthetics canary checking public redirect latency and API health periodically.

---

## 4. API & Data Model Enhancements

- [ ] **Batch Link Creation Endpoint (`POST /api/batch`)**
  - Support creating multiple short links in a single request using DynamoDB `BatchWriteItem` via direct API Gateway VTL mapping.
- [ ] **Link Soft-Delete & Archive State**
  - Implement a `status` attribute (`ACTIVE`, `DISABLED`, `ARCHIVED`) with conditional checks in the `GetItem` VTL template to allow disabling links without permanently removing records.
- [ ] **Custom Metadata & Tags**
  - Allow storing arbitrary string metadata tags (e.g. campaign ID, department) alongside link records for organizational tracking.
