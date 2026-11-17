# Functionless URL Shortener

[![Latest Release](https://img.shields.io/github/v/release/divmora/functionless-url-shortener?logo=github)](https://github.com/divmora/functionless-url-shortener/releases)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI/CD](https://github.com/divmora/functionless-url-shortener/actions/workflows/ci.yml/badge.svg)](https://github.com/divmora/functionless-url-shortener/actions)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/divmora/functionless-url-shortener)
[![Security Policy](https://img.shields.io/badge/Security-Policy-green.svg)](SECURITY.md)

A high-performance, cost-effective URL shortener built completely without compute (no AWS Lambda functions). All routing and business logic are handled directly at the **Amazon API Gateway** layer via Velocity Template Language (VTL) mapping templates that communicate natively with **Amazon DynamoDB**.

The administrative endpoints are secured using **AWS IAM Authentication (SigV4)** and restricted to accounts within a specific **AWS Organization** via API Gateway Resource Policies. Public URL redirection is accelerated through **Amazon CloudFront**.

---

## Architecture & Features

* **Zero Compute (Functionless):** Direct API Gateway service integration to DynamoDB (`GetItem` for lookups, `UpdateItem` with conditional checks for creating links).
* **IAM & AWS Organizations Security:** Management endpoints (`/api/*`) require AWS SigV4 signed requests and are restricted to identities belonging to your specified AWS Organization ID (`aws:PrincipalOrgID`).
* **Public High-Speed Redirects:** Public endpoints (`/{linkId}` and `/?go={linkId}`) perform fast HTTP 301 redirects directly from DynamoDB lookups, cached at the edge with CloudFront.
* **Automatic Expiration (TTL):** Built-in DynamoDB Time-To-Live support automatically cleans up expired links.
* **Traffic Dashboard:** An optional Amazon CloudWatch dashboard (`EnableDashboard`, on by default) showing CloudFront requests and error rates, API Gateway volume and latency, DynamoDB capacity and throttles, and alarm status. Its link is in the `DashboardUrl` stack output.
* **Observability & Alerting:** Comprehensive Amazon CloudWatch alarms configured for API Gateway (4xx, 5xx, p99 Latency), DynamoDB (User/System Errors), and CloudFront (Error Rate, Cache Hit Rate), notifying an Amazon SNS topic.

---

## Services Used

* [Amazon API Gateway](https://aws.amazon.com/api-gateway/) – REST API with VTL direct service integrations and resource policy authorization.
* [Amazon DynamoDB](https://aws.amazon.com/dynamodb/) – Fast NoSQL key-value store with TTL and GSI support.
* [Amazon CloudFront](https://aws.amazon.com/cloudfront/) – Edge CDN distribution for fast global redirects and caching.
* [Amazon CloudWatch](https://aws.amazon.com/cloudwatch/) – Operational metrics and alarms.
* [Amazon SNS](https://aws.amazon.com/sns/) – Notification topic for triggered alarms.

---

## Requirements for Deployment

* [AWS CLI](https://aws.amazon.com/cli/) (configured with appropriate deployment credentials)
* [AWS SAM CLI v0.37.0+](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
* **Make** (for automated build and validation workflows)

---

## Deployment

### Guided Deployment

For the first deployment, run the guided SAM deploy command:

```bash
make deploy-guided
```

Or deploy specifying a configuration environment:

```bash
sam deploy -g --config-file samconfig.toml --config-env prd
```

### Configuration Parameters

During the guided deployment, SAM CLI will prompt for the following parameters:

| Parameter | Required | Default | Description |
| :--- | :--- | :--- | :--- |
| **Stack Name** | Yes | `URLShortener` | The name of the CloudFormation stack. |
| **AWS Region** | Yes | `ap-south-1` | The AWS region to deploy into. |
| **AppName** | No | `shortener` | Application name prefix (must be globally unique, no spaces). |
| **OrgId** | **Yes** | *(None)* | Your AWS Organization ID (e.g. `o-xxxxxxxxxx`) used to restrict IAM access on `/api/*`. |
| **CustomDomainName** | No | `""` | Optional URL prefix used to format short URLs returned by the API (e.g. `https://sho.rt`). Defaults to `https://<DomainAlias>` when `DomainAlias` is set. |
| **DomainAlias** | No | `""` | Optional custom hostname served by CloudFront (e.g. `go.example.com`). Requires `AcmCertificateArn`. |
| **AcmCertificateArn** | Conditional | `""` | ACM certificate ARN in **us-east-1** covering `DomainAlias`. Required when `DomainAlias` is set. |
| **EnableDashboard** | No | `true` | Create a CloudWatch dashboard for link traffic and service health. |
| **AlarmEmail** | No | `""` | Email address subscribed to alarm notifications (confirm the SNS email after deploy). |
| **EnableCloudFrontAdditionalMetrics** | No | `false` | Enable CloudFront additional metrics (needed for `CacheHitRate` and its alarm); billed as CloudWatch custom metrics. |
| **HostedZoneId** | No | `""` | Optional Route 53 hosted zone ID. When set, alias `A`/`AAAA` records for `DomainAlias` are created automatically. |
| **TTL** | No | `31536000` | Default URL lifespan in seconds (default is 1 year). |

#### Example Interactive Prompt

```bash
## The name of the CloudFormation stack
Stack Name [URLShortener]: 

## The region you want to deploy in
AWS Region [ap-south-1]: 

## The name of the application (lowercase no spaces). This must be globally unique
Parameter AppName [shortener]: 

## AWS OrgId to protect APIs using IAM Auth to only accounts related to AWS Org
Parameter OrgId []: o-xxxxxxxxxx

## Domain Name for the shortener (optional)
Parameter CustomDomainName []: https://sho.rt

## TTL time for url to be active in seconds
Parameter TTL [31536000]: 

## Shows you resource changes to be deployed and requires a 'Y' to initiate deploy
Confirm changes before deploy [y/N]: y

## SAM needs permission to be able to create roles to connect to the resources in your template
Allow SAM CLI IAM role creation [Y/n]: Y

## Save your choices for later deployments
Save arguments to samconfig.toml [Y/n]: Y
```

> [!NOTE]
> Creating or updating the Amazon CloudFront distribution may take a few minutes. Subsequent deployments with unchanged CloudFront settings will be significantly faster.

After the initial setup, you can re-deploy anytime using:

```bash
make deploy
```

---

## Custom Domain

To serve short links from your own domain (e.g. `https://go.example.com/abc123`):

1. **Request a certificate in `us-east-1`.** CloudFront only accepts ACM certificates from `us-east-1`, regardless of the stack's region:
   ```bash
   aws acm request-certificate --region us-east-1 \
     --domain-name go.example.com --validation-method DNS
   ```
   Complete DNS validation and wait until the certificate status is `ISSUED`.
2. **Deploy with the custom domain parameters:**
   ```bash
   sam deploy --parameter-overrides \
     OrgId=o-xxxxxxxxxx \
     DomainAlias=go.example.com \
     AcmCertificateArn=arn:aws:acm:us-east-1:111122223333:certificate/xxxx \
     HostedZoneId=Z0123456789ABCDEFGHIJ   # optional
   ```
3. **Point DNS at CloudFront.** If `HostedZoneId` was provided, Route 53 alias records are created for you. Otherwise, create a `CNAME` for `DomainAlias` pointing to the `DnsTarget` stack output.

The `ShortUrlBase` stack output shows the base URL that short links are returned with.

---

## Monitoring & Alarms

| Alarm | Metric | Fires when |
| :--- | :--- | :--- |
| `<Stack> API 4xx` / `API 5xx` | `AWS/ApiGateway` `4XXError` / `5XXError` (`ApiName=<Stack>-api`, `Stage=Prod`) | Error rate (Average) > 1% over 1 min |
| `<Stack> API Latency` | `AWS/ApiGateway` `Latency` | p99 > 75 ms over 5 min |
| `<Stack> DynamoDB SystemErrors` | `AWS/DynamoDB` `SystemErrors` summed over `GetItem`, `UpdateItem`, `DeleteItem` | ≥ 1 error/min in 2 of 5 minutes |
| `<Stack> CloudFront Error Rate` | `AWS/CloudFront` `TotalErrorRate` (`Region=Global`) | Average > 5% over 5 min |
| `<Stack> CloudFront Cache Hit Rate` *(opt-in)* | `AWS/CloudFront` `CacheHitRate` (additional metric) | Average ≤ 80% for 15 min |

Regional alarms notify the `AlarmTopicArn` output topic; set `AlarmEmail` to subscribe an address. Missing data (no traffic) is treated as OK.

### CloudFront alarms (us-east-1)

CloudFront publishes metrics only in **us-east-1**, so its alarms live in [`cloudfront-alarms.yaml`](cloudfront-alarms.yaml), with their own SNS topic in that region.

* **Main stack in `us-east-1`:** created automatically as a nested stack, notifying the main topic.
* **Main stack in any other region:** deploy the alarms yourself in `us-east-1`:
  ```bash
  aws cloudformation deploy --region us-east-1 \
    --stack-name URLShortener-cloudfront-alarms \
    --template-file cloudfront-alarms.yaml \
    --parameter-overrides \
      DistributionId=<CloudFrontDistributionId output> \
      AlarmNamePrefix=URLShortener \
      AlarmEmail=ops@example.com \
      EnableCacheHitRateAlarm=false
  ```
  Set `EnableCacheHitRateAlarm=true` only if the main stack has `EnableCloudFrontAdditionalMetrics=true`.

---

## Admin CLI (`shorten`)

`shorten` signs management API requests for you using the standard AWS credential chain (environment variables, shared profiles, IAM Identity Center / SSO, instance roles), so you never have to hand-craft SigV4 requests.

### Install

Requires Go 1.26+:

```bash
go install github.com/divmora/functionless-url-shortener/cmd/shorten@latest
```

### Usage

```bash
# Create a link (a random 7-character ID is generated if --id is omitted)
shorten create https://example.com/very/long/path --id launch --ttl 604800

# Look up a link
shorten get launch

# Delete a link
shorten delete launch

# JSON output for scripting
shorten get launch --output json
```

The CLI reads the management endpoint from the `ManagementApiEndpoint` output of the CloudFormation stack (default stack name `URLShortener`). Override with:

| Flag | Env var | Description |
| :--- | :--- | :--- |
| `--stack` | `SHORTEN_STACK` | Stack to read the endpoint from (default `URLShortener`). |
| `--endpoint` | `SHORTEN_ENDPOINT` | Management API endpoint; skips the stack lookup. |
| `--region` | `AWS_REGION` | AWS region (inferred from `--endpoint` when possible). |
| `--profile` | `AWS_PROFILE` | AWS shared config profile, e.g. an IAM Identity Center profile. |
| `--output` | | `text` (default) or `json`. |

Exit codes: `0` success, `1` error, `2` usage error, `3` link not found, `4` link ID already exists.

> [!NOTE]
> The CLI calls the API Gateway endpoint directly rather than CloudFront: CloudFront rewrites the `Host` header, which invalidates SigV4 signatures. Looking up the endpoint from the stack additionally requires `cloudformation:DescribeStacks`; use `--endpoint` if the caller lacks it.

---

## IAM Permissions

Clients calling the secured management endpoints (`/api/*`) require an IAM principal belonging to the configured AWS Organization with the following least-privilege policy:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowURLShortenerManagementAPI",
            "Effect": "Allow",
            "Action": [
                "execute-api:Invoke"
            ],
            "Resource": [
                "arn:aws:execute-api:*:*:*/*/POST/api",
                "arn:aws:execute-api:*:*:*/*/GET/api/*",
                "arn:aws:execute-api:*:*:*/*/DELETE/api/*"
            ]
        }
    ]
}
```

---

## API Reference

### 1. Create a Short URL

Creates a new short URL record in DynamoDB. Returns an error if the ID already exists.

* **Endpoint:** `POST /api`
* **Authentication:** AWS SigV4 (IAM) + AWS Organization check (`aws:PrincipalOrgID`)
* **Headers:**
  * `Content-Type: application/json`
  * `Authorization: AWS4-HMAC-SHA256 ...`

#### Request Body
```json
{
  "id": "my-custom-slug",
  "url": "https://example.com/very/long/destination/url",
  "ttl": 604800
}
```
* `id` *(string, required)*: The unique short identifier.
* `url` *(string, required)*: The destination URL (must start with `http://` or `https://`).
* `ttl` *(integer, optional)*: Custom time-to-live in seconds from the current time. If omitted, uses stack default.

#### Response (`200 OK`)
```json
{
  "id": "my-custom-slug",
  "url": "https://example.com/very/long/destination/url",
  "shortUrl": "https://sho.rt/?go=my-custom-slug",
  "timestamp": "2026-08-20T08:30:00.000Z",
  "owner": "AROAEXAMPLEUSER:session-name"
}
```

#### Error Response (`422 Unprocessable Entity`)
Returned if the `id` already exists in DynamoDB:
```json
{
  "error": true,
  "message": "URL link already exists"
}
```

---

### 2. Get Link Details

Fetches metadata for an existing short link.

* **Endpoint:** `GET /api/{linkId}`
* **Authentication:** AWS SigV4 (IAM) + AWS Organization check (`aws:PrincipalOrgID`)
* **Response (`200 OK`):**
```json
{
  "id": "my-custom-slug",
  "url": "https://example.com/very/long/destination/url",
  "shortUrl": "https://sho.rt/?go=my-custom-slug",
  "timestamp": "2026-08-20T08:30:00.000Z",
  "owner": "AROAEXAMPLEUSER:session-name"
}
```

---

### 3. Delete a Short URL

Deletes an existing short link and returns the deleted record.

* **Endpoint:** `DELETE /api/{linkId}`
* **Authentication:** AWS SigV4 (IAM) + AWS Organization check (`aws:PrincipalOrgID`)
* **Response (`200 OK`):**
```json
{
  "id": "my-custom-slug",
  "url": "https://example.com/very/long/destination/url",
  "owner": "AROAEXAMPLEUSER:session-name"
}
```
* **Response (`404 Not Found`)** if the link does not exist:
```json
{ "error": true, "message": "URL link not found" }
```

> [!NOTE]
> CloudFront caches redirects for up to 5 minutes (`Cache-Control: max-age=300`), so a deleted link may keep redirecting briefly unless you create a CloudFront invalidation.

---

### 4. URL Redirection (Public)

Performs an HTTP 301 redirect to the destination URL. No authentication required.

* **Endpoints:**
  * `GET /{linkId}`
  * `GET /?go={linkId}`
* **Responses:**
  * `301 Moved Permanently`: Redirects to the registered `url` with `Location: <destination_url>`.
  * If the link ID does not exist, redirects to `Location: error?error=url_not_found`.

---

## Development & Building

### Common Commands

```bash
# Validate SAM template and OpenAPI definitions
make validate

# Build SAM application
make build

# Guided interactive deployment
make deploy-guided

# Deploy with existing samconfig.toml
make deploy

# Run Go unit tests for the shorten CLI
make test

# Build the shorten CLI into ./bin/shorten
make cli

# Clean temporary build artifacts
make clean
```

---

## Cleanup

To remove all deployed resources:

1. Using SAM CLI:
   ```bash
   sam delete --stack-name URLShortener
   ```
2. Or via the AWS Management Console:
   * Open the [AWS CloudFormation Console](https://console.aws.amazon.com/cloudformation/home).
   * Select the stack named **URLShortener** (or your custom stack name).
   * Click **Delete** and confirm.

---

## Community & Contributing

- **[Contributing Guide](CONTRIBUTING.md)**: Review contribution guidelines, development workflows, and Conventional Commit requirements.
- **[Product Roadmap](ROADMAP.md)**: View planned architectural vision, edge optimizations, and upcoming capabilities.
- **[Code of Conduct](https://github.com/divmora/.github/blob/main/CODE_OF_CONDUCT.md)**: Contributor Covenant Code of Conduct.
- **[Security Policy](SECURITY.md)**: Guidelines for reporting security vulnerabilities responsibly.

---

## License

This project is licensed under the **Apache License, Version 2.0**. See the [LICENSE](LICENSE) file for details.
