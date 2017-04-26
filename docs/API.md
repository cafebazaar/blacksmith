## APIs

- HTTPS Swagger API used by blacksmithctl and blacksmith-agent server. See (swagger.yaml)[https://github.com/cafebazaar/blacksmith/blob/dev/swagger.yaml].
- HTTP API for serving Cloudconfig, ignition, bootparams files.
  - GET /t/cc/$MAC
  - GET /t/ig/$MAC
  - GET /t/bp/$MAC
