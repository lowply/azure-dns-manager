## This repository has been archvied.

Azure DNS Manager was a learning project for me, and not maintained anymore. The best alternative is [octodns/octodns: Tools for managing DNS across multiple providers](https://github.com/octodns/octodns).

---

# Azure DNS Manager

Simple DNS manager for Azure DNS.

## Install

```
go get -u github.com/lowply/azure-dns-manager
```

Note that installing it with `go get` will take some time.

## Prep

Create the config dir

```
mkdir -p ~/.config/azure-dns-manager
```

Generate the Azure auth file

```
az ad sp create-for-rbac -n "azure-dns-manager" --role contributor --sdk-auth true > ~/.config/azure-dns-manager/auth.json
```

Export `AZURE_AUTH_LOCATION` and `AZURE_DNS_ZONES` env vars. I keep my zone files in the `lowply/dns` repository.

```
export AZURE_AUTH_LOCATION=${HOME}/.config/azure-dns-manager/auth.json
export AZURE_DNS_ZONES=${HOME}/.ghq/github.com/lowply/dns/zones
```

## Usage


Sync a zone YAML file to Azure DNS

```
azure-dns-manager -s example.com
```

Get zone data from an existing zone and output it to STDOUT in YAML format

```
azure-dns-manager -g example.com
```

Redirect it to a file

```
azure-dns-manager -g example.com > /path/to/example.com.yaml
``` 

Getting NS records for a domain

```
azure-dns-manager -ns example.com
```

## Zone YAML format example

The YAML file should be written in the following format.

```yaml
Zone: example.com # Zone name
RecordSets:
- Name: '@' # Use @ for apex records. Special characters like @ need to be quoted
  Type: A # Type should be one of the following: A, AAAA, CNAME, MX, NS, TXT
  Properties: # TTL and values should be part of Properties
    TTL: 300
    Values: # Multiple values are allowed
    - 52.69.40.184
- Name: www # Non-special characters don't need to be quoted
  Type: A
  Properties:
    TTL: 300
    Values:
    - 52.69.40.184
- Name: '@'
  Type: NS
  Properties:
    TTL: 3600 # Only TTL can be configurable for NS records
- Name: '@'
  Type: MX
  Properties:
    TTL: 300
    Values: # Preference and value for MX records should be separated with a single space
    - 20 ALT1.ASPMX.L.GOOGLE.COM.
    - 10 ASPMX.L.GOOGLE.COM.
    - 20 ALT2.ASPMX.L.GOOGLE.COM.
    - 30 ASPMX2.GOOGLEMAIL.COM.
    - 30 ASPMX3.GOOGLEMAIL.COM.
    - 30 ASPMX4.GOOGLEMAIL.COM.
    - 30 ASPMX5.GOOGLEMAIL.COM.
- Name: '@'
  Type: TXT
  Properties:
    TTL: 300
    Values:
    - v=spf1 include:_spf.google.com ~all # No escape sequences and quotes are needed for TXT records
- Name: google._domainkey # Name should not include the domain (example.com in this example)
  Type: TXT
  Properties:
    TTL: 300
    Values: # Can be longer than 255 characters for a TXT record. azure-dns-manager will split it into multiple values on Azure DNS
    - v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqkrDM1GCBFzxlZAzwzgdGp4s4cEZUCA0o0uJFlflQvW05VYvTFocf3Zfp9QCT4lsPoBrAPjrLc3RzroAkvDmNCNCJPg/U7mwuJRg+wYF/qHy6Dlp7djsXzOY833PjIMBBfZsMuF8HHsPmvvLSbWlCft4rscV8vV185/5idR0wUmZEcmmG2SJJxJMC667465s8s4wONFR5lsTOqVMCZ0TRKnBB2XbexfdzXNFdOkwF+V1XBNoNMNVKrcyJDb16JR5omfQRcIjV3sFAdPQ5DMwfCR/qcshW+33b4xOHh85+V5N+cnzEVzQqLm+lwDZnIehkSL6nvKmIwqg/w6Epk7FTwIDAQAB
```

Please refer to [Overview of DNS zones and records](https://docs.microsoft.com/en-us/azure/dns/dns-zones-records) for Azure DNS specs.

## Note

- Doesn't touch SOA and NS records for safety reason (Only TTL can be changed for NS records)
- Doesn't support PTR and SRV records
- Alias support will come soon (Ref: [Announcing Alias records for Azure DNS | Blog | Microsoft Azure](https://azure.microsoft.com/en-us/blog/announcing-alias-records-for-azure-dns/))
