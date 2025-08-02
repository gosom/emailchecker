# Email Checker

A Go tool for validating and analyzing email addresses to detect disposable emails, suspicious patterns, and assess deliverability.

## What it does

- **Validates email addresses** - checks format, DNS records, and deliverability
- **Detects disposable emails** - identifies temporary/throwaway email providers  
- **Analyzes email patterns** - finds suspicious or bot-generated emails
- **Checks domain reputation** - verifies against well-known and educational domains
- **Risk scoring** - provides low/medium/high risk assessment with detailed reasoning

## Try it online

Use the UI at: `https://emailcheck-api.thexos.dev/`
Test the API at: `https://emailcheck-api.thexos.dev/check/{email}`

Example: https://emailcheck-api.thexos.dev/check/test@gmail.com

(some rate limits apply)

## Features

- Fast concurrent processing for bulk email validation
- SQLite database for caching DNS records and domain lists
- Automatic updates for disposable email and domain lists (every 13 hours)
- Risk analysis with detailed reasoning
- Educational domain detection for universities and schools
- Pattern analysis to detect automated/bot registrations
- Parked domain detection to identify inactive domains
- HTTP API with JSON responses

## Installation

1. Install Go 1.24+ 
2. Clone this repository
3. Run: `go mod tidy`
4. Build: `go build -o checker cmd/email-checker/main.go`

## Environment Variables


- EMAIL_CHECKER_DB_PATH - Path to SQLite database file (default: checker.db)
- ALLOWED_HOSTS - Comma-separated list of allowed hosts for API (default: localhost:8080)


## Usage

### Check single email

```bash
./checker check user@example.com
```

### Check multiple emails from file

```bash
./checker check --file emails.txt
```

### Check emails from stdin

```bash
echo "user@example.com" | ./checker check --stdin
```

### Start HTTP server

```bash
./checker server --port :8080
```

OR

#### Docker

```bash
docker build -t email-checker .
docker run -p 8080:8080 email-checker
```


Then check emails via HTTP:

```bash
curl "http://localhost:8080/check/user@example.com"
```

### Update database manually

```bash
./checker update
```

### Example Output

```json
{
  "email": "test@forexzig.com",
  "disposable": {
    "checked": true,
    "value": true,
    "error": null,
    "elapsed": 4650774
  },
  "well_known": {
    "checked": true,
    "value": false,
    "error": null,
    "elapsed": 1221592
  },
  "educational": {
    "checked": true,
    "value": false,
    "error": null,
    "elapsed": 3656926
  },
  "dns": {
    "checked": true,
    "value": {
      "domain": "forexzig.com",
      "has_mx": true,
      "has_spf": false,
      "has_dmarc": false,
      "is_parked": false,
      "a_records": [
        "172.67.177.120",
        "104.21.75.139"
      ],
      "ns_records": [
        "adel.ns.cloudflare.com.",
        "nitin.ns.cloudflare.com."
      ],
      "mx_records": [
        {
          "value": "mx2.den.yt.",
          "priority": 10,
          "disposable": true
        }
      ],
      "spf_record": "",
      "dmarc_record": ""
    },
    "error": null,
    "elapsed": 35069721
  },
  "elapsed": 35131609,
  "pattern": {
    "checked": true,
    "value": {
      "short_local_part": false,
      "has_random_pattern": false,
      "too_many_consecutive_numbers": false,
      "too_many_special_chars": false
    },
    "error": null,
    "elapsed": 28608
  },
  "prediction": {
    "risk_level": "high",
    "score": 1,
    "reasons": [
      "Disposable email provider blocked"
    ]
  }
}
```
