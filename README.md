# lgtmp-query-gateway

**lgtmp-query-gateway** is a gateway designed to enforce **Role-Based Access Control (RBAC)** and **Label-Based Access Control (LBAC)** for queries originating from Grafana. It integrates with **Log**, **Grafana**, **Tempo**, **Mimir**, and **Pyroscope** to provide secure and controlled access to data.

## Features

- **RBAC Enforcement**: Validates user roles based on OIDC token claims to ensure only authorized users can access specific resources.
- **LBAC Enforcement**: Restricts query results by applying label-based filters to enforce fine-grained access control.
- **OIDC Integration**: Leverages OIDC token claims to determine user group memberships and apply access restrictions accordingly.
- **Multi-Tenant Support**: Designed to handle multiple tenants with distinct access control rules.

## How It Works

1. **Query Interception**: The gateway intercepts queries sent to Grafana.
2. **Token Validation**: It validates the user's OIDC token to extract group membership claims.
3. **Access Control**: Based on the user's groups and the configured RBAC/LBAC rules, the gateway enforces restrictions on the query.
4. **Query Forwarding**: If the query passes all checks, it is forwarded to the appropriate backend (e.g., Loki, Tempo, Mimir).

## Supported Stacks

1. **Loki**
2. **Mimir**
3. **Tempo** (Planned)
4. **Pyroscope** (Planned)

## Supported OIDC Providers

1. **EntraID**

## Project Status

🚧 **This project is under heavy development and is not ready for production use.** 🚧  
Breaking changes, incomplete features, and potential security issues are expected at this stage. Use it only for testing and development purposes.

## Getting Started

### Prerequisites

- Go 1.20+
- Docker (optional, for containerized deployment)
- An OIDC provider (e.g., Keycloak, Auth0)

### Installation

Clone the repository:

```bash
git clone https://github.com/your-repo/lgtmp-query-gateway.git
cd lgtmp-query-gateway
```

Build the project:
```base
go build ./cmd/gateway
```

### Configuration

The gateway is configured via a config.yaml file. Example configuration:
```yaml
"<vhost>": # hostname that the gateway will listen for
  type: "loki" # loki|mimir|prometheus|tempo|pyroscope
  upstream: "http://localhost:9001" # where should the gateway send requests after validations
  allowUndefined: true # allow access to undefined tenants
  tenants:
    test:
      mode: "allowlist" # allow or deny (denylist) access from the following groups
      groups:
        - name: "group1"
        - name: "group2"
          enforcedLabels: # allow access but enforce the use of these label selectors
            - 'sensitive!="true"'
            - 'source="kubernetes"'
```

## Running the Gateway

Start the gateway:
```bash
./gateway -t <tenantid> -c <clientid> -f config.yaml
```
