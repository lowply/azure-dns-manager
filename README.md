# Azure DNS Manager

Simple DNS manager for Azure DNS.

## Install

```
go get github.com/lowply/azure-dns-manager
```

Note that installing it with `go get` will take some time.

## Prep

```
mkdir -p ~/.config/azure-dns-manager
az ad sp create-for-rbac -n "azure-dns-manager" --role contributor --sdk-auth true > ~/.config/azure-dns-manager/auth.json
```

## Usage

Export `AZURE_AUTH_LOCATION` first

```
export AZURE_AUTH_LOCATION=${HOME}/.config/azure-dns-manager/auth.json
```

Sync from zone file to Azure DNS

```
azure-dns-manager -s example.com
```

Generate a zone file from an existing zone in Azure DNS

```
azure-dns-manager -g example.com -p /path/to/example.com.yaml
```

When the `-p` flag is omitted, the `example.com_remote.yaml` file will be saved in the `~/.config/azure-dns-manager/zones` directory.

```
azure-dns-manager -g example.com
```

## Note

- Doesn't touch SOA and NS records for safety reason (TTL can be changed for NS records)
- Doesn't support CAA, PTR and SRV records
- Alias support will come soon (Ref: [Announcing Alias records for Azure DNS | Blog | Microsoft Azure](https://azure.microsoft.com/en-us/blog/announcing-alias-records-for-azure-dns/))
