# Email Checker

A Go tool for validating and analyzing email addresses.

This is still Work In Progress

## What it does

- **Validates email addresses** - checks format, DNS records, and deliverability
- **Detects disposable emails** - identifies temporary/throwaway email providers  
- **Analyzes email patterns** - finds suspicious or bot-generated emails
- **Checks domain reputation** - verifies against well-known and educational domains
- **Risk scoring** - provides low/medium/high risk assessment

## Installation

1. Install Go 1.24+ 
2. Clone this repository
3. Run: `go mod tidy`
4. Build: `go build -o checker cmd/email-checker/main.go`

## Usage

### Check single email

```bash
./checker check user@example.com
```

### Check multiple emails from file

```bash
./checker check --file emails.txt
```

### Start HTTP server

```bash
./checker server --port 8080
```

Then check emails via HTTP:

```bash
curl "http://localhost:8080/check?email=user@example.com"
```

### Example Output

```json
{
  "Email": "jakezopa@forexzig.com",
  "Disposable": {
    "Checked": true,
    "Value": true,
    "Err": null,
    "Elapsed": 560441
  },
  "WellKnown": {
    "Checked": true,
    "Value": false,
    "Err": null,
    "Elapsed": 465260
  },
  "Educational": {
    "Checked": true,
    "Value": false,
    "Err": null,
    "Elapsed": 372870
  },
  "DNS": {
    "Checked": true,
    "Value": {
      "Domain": "forexzig.com",
      "HasMX": true,
      "HasSPF": false,
      "HasDMARC": false,
      "MXRecords": [
        {
          "Value": "mx2.den.yt.",
          "Priority": 10,
          "Disposable": true
        }
      ],
      "SPFRecord": "",
      "DMARCRecord": ""
    },
    "Err": null,
    "Elapsed": 39309592
  },
  "Elapsed": 39527786,
  "Pattern": {
    "Checked": true,
    "Value": {
      "ShortLocalPart": false,
      "HasRandomPattern": false,
      "TooManyConsecutiveNumbers": false,
      "TooManySpecialChars": false
    },
    "Err": null,
    "Elapsed": 122612
  },
  "Analysis": {
    "risk_level": "high",
    "score": 1,
    "reasons": [
      "Disposable email provider blocked"
    ]
  }
}
```

## Features

* **Fast concurrent processing** for bulk email validation
* **SQLite database** for caching DNS records and domain lists
* **Automatic updates** for disposable email and domain lists
* **Risk analysis** with detailed reasoning
* **Educational domain detection** for universities and schools
* **Pattern analysis** to detect automated/bot registrations
