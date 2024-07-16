# snctl

A tool to automate maintenance / content tasks around the startup nights 
website. See `snctl --help` for more information.

```sh
snctl token update --drive --gmail --sheets --update-secrets
```

## Notes

For triggering the github action, an access token is required with the 
following permissions:

* actions: read/write
* environments: read/write
* variables: read/write
