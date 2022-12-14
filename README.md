# Client Registrar

The client registrar was split from gitlab.com/elixxir/registration to create a standalone service to handle client registration

## Example Configuration File

```yaml
# ==================================
# Client Registrar Configuration
# ==================================

# Log message level (0 = info, 1 = debug, >1 = trace)
logLevel: 1
# Path to log file
logPath: "registration.log"

# Public address, used in NDF it gives to client
publicAddress: "0.0.0.0:11420"
# The listening port of this server
port: 11420

# === REQUIRED FOR ENABLING TLS ===
# Path to the permissioning server private key file
keyPath: ""
# Path to the permissioning server certificate file
certPath: ""
# Path to the signed registration server private key file
signedKeyPath: ""
# Path to the signed registration server certificate file
signedCertPath: ""

# Maximum number of connections per period
userRegCapacity: 1000
# How often the number of connections is reset
userRegLeakPeriod: "24h"

# Database connection information
dbUsername: "cmix"
dbPassword: ""
dbName: "cmix_server"
dbAddress: ""

# List of client codes to be added to the database (for testing)
clientRegCodes:
  - "AAAA"
  - "BBBB"
  - "CCCC"
```
