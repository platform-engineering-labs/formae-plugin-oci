# OCI plugin for formae

[![CI](https://github.com/platform-engineering-labs/formae-plugin-oci/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-oci/actions/workflows/ci.yml)
[![Monthly](https://github.com/platform-engineering-labs/formae-plugin-oci/actions/workflows/monthly.yml/badge.svg?branch=main)](https://github.com/platform-engineering-labs/formae-plugin-oci/actions/workflows/monthly.yml)

Manages Oracle Cloud Infrastructure (OCI) VCNs, subnets, compute instances, OKE clusters, object storage and IAM as Infrastructure As Code with [formae](https://github.com/platform-engineering-labs/formae).

[formae](https://github.com/platform-engineering-labs/formae) · [Hub](https://hub.platform.engineering/platform.engineering/oci) · [Configuration](https://docs.formae.ai/documentation/reference/providers/oci/configuration) · [Supported resources](https://docs.formae.ai/documentation/reference/providers/oci/supported-resources)

## Install

Requires the formae CLI: see the [quick start](https://docs.formae.ai/documentation/get-started/quickstart).

```bash
formae plugin install oci
```

Restart the formae agent afterwards so it loads the plugin.

**New project:** with the agent running, `formae project init --include oci my-project` creates `my-project` with a `PklProject` that declares the formae and oci schema packages, so `import "@oci/..."` resolves, and a starter `main.pkl`. Don't run it in an existing project: it overwrites both files.

**Existing project:** add the plugin to `dependencies` in your `PklProject`, with the current version from the [hub page](https://hub.platform.engineering/platform.engineering/oci), then run `pkl project resolve`:

```pkl
["oci"] {
  uri = "package://hub.platform.engineering/plugins/oci/schema/pkl/oci/oci@<version>"
}
```

Next: [write your first forma](https://docs.formae.ai/documentation/get-started/write-your-first-forma), then [`formae apply`](https://docs.formae.ai/documentation/reference/cli/apply) (see [apply modes](https://docs.formae.ai/documentation/concepts/apply-modes)).

With an AI coding assistant, use the [formae plugin](https://docs.formae.ai/documentation/guides/ai-coding-assistants) (formerly `formae-mcp`), which can search the hub and fetch plugin examples. The formae documentation is also available as [llms.txt](https://docs.formae.ai/llms.txt).

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `OCI::Identity::Compartment` | Compartments |
| `OCI::Core::Vcn` | Virtual Cloud Networks |
| `OCI::Core::Subnet` | Subnets |
| `OCI::Core::InternetGateway` | Internet gateways |
| `OCI::Core::NatGateway` | NAT gateways |
| `OCI::Core::ServiceGateway` | Service gateways |
| `OCI::Core::RouteTable` | Route tables |
| `OCI::Core::SecurityList` | Security lists |
| `OCI::Core::NetworkSecurityGroup` | Network security groups |
| `OCI::Core::NetworkSecurityGroupSecurityRule` | NSG security rules |
| `OCI::Core::DhcpOptions` | DHCP options |
| `OCI::Core::Instance` | Compute instances |
| `OCI::Core::Volume` | Block volumes |
| `OCI::Identity::Policy` | IAM policies |
| `OCI::ContainerEngine::Cluster` | OKE clusters |
| `OCI::ContainerEngine::NodePool` | OKE node pools |
| `OCI::ContainerEngine::VirtualNodePool` | OKE virtual node pools |
| `OCI::ObjectStorage::Bucket` | Object storage buckets |

## Configuration

Configure an OCI target in your Forma file:

```pkl
import "@oci/oci.pkl"

new formae.Target {
    label = "my-oci-target"
    config = new oci.Config {
        region = "us-ashburn-1"
        profile = "DEFAULT"
    }
}
```

Authentication uses the OCI SDK's default config provider:
- Config file (`~/.oci/config`)
- Environment variables
- Instance principal (on OCI compute)

## Examples

See [examples/](examples/) for usage patterns:

- `lifeline/` - VCN networking infrastructure
- `oke/` - OKE Kubernetes cluster

**Note:** Update `vars.pkl` with your compartment ID and region before running.

## License

FSL-1.1-ALv2
