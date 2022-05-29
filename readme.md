Paste this to [zerops.io](https://app.zerops.io) as import services yml:
```yaml
services:
  - hostname: lifeline
    type: go@1
    ports:
      - port: 1999
    envVariables:
      GOOGLE_JSON_CREDENTIALS: |-
        <your-service-account-credentials-json-content>
      SPREADSHEET_ID: <your-spreadsheet-id>
    buildFromGit: https://github.com/tikinang/lifeline
    enableSubdomainAccess: true
    minContainers: 1
    maxContainers: 1
```