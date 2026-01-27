# OKE Cluster Infrastructure with Pkl

This example demonstrates how to provision an Oracle Kubernetes Engine (OKE) cluster on OCI with a production-ready network configuration.

## Files

- `/opt/pel/formae/examples/complete/oke-example/main.pkl` - Main infrastructure entry point
- `/opt/pel/formae/examples/complete/oke-example/vars.pkl` - Configuration variables
- `/opt/pel/formae/examples/complete/oke-example/infrastructure/vcn.pkl` - VCN configuration
- `/opt/pel/formae/examples/complete/oke-example/infrastructure/network.pkl` - Network components (gateways, route tables, subnets)
- `/opt/pel/formae/examples/complete/oke-example/infrastructure/security.pkl` - Security list with OKE rules
- `/opt/pel/formae/examples/complete/oke-example/infrastructure/oke.pkl` - OKE cluster and node pool configuration

## What Gets Created

- VCN with public and private subnets
- Internet Gateway for public subnet access
- NAT Gateway for private subnet egress to internet
- Service Gateway for OCI services access
- Route Tables for public and private subnets
- Security List with OKE-required rules
- OKE Cluster with public API endpoint
- Node Pool with flexible VM shapes (E4.Flex)

## Configuration

Edit the configuration values in `vars.pkl`:

```pkl
projectName = "oke-example"
ociRegion = "us-chicago-1"
k8sVersion = "v1.34.1"

// Network configuration
vcnCidr = "10.0.0.0/16"
serviceLbSubnetCidr = "10.0.1.0/24"
workerNodeSubnetCidr = "10.0.2.0/24"

// Node pool configuration
vmShape = "VM.Standard.E4.Flex"
nodeOcpus = 2.0
nodeMemoryGBs = 16.0
nodePoolSize = 2
```

## Prerequisites

Before deploying, you need region-specific values:

1. **Node Image OCID** - Find OKE-optimized images for your region:
   ```bash
   oci ce node-pool-options get \
     --node-pool-option-id all \
     --compartment-id <compartment-id> \
     --profile <oci-profile>
   ```

2. **Service Gateway Service ID** - Find the service ID for your region:
   ```bash
   oci network service list \
     --compartment-id <compartment-id> \
     --profile <oci-profile> \
     --query "data[?contains(name, 'Services')].{Name:name, Id:id}" \
     --output table
   ```

## Usage

1. Ensure the **formae** node is up and running.

2. Deploy to OCI:
   ```bash
   formae apply --mode reconcile main.pkl
   ```

   Or with a custom compartment:
   ```bash
   formae apply --mode reconcile main.pkl --compartmentId <your-compartment-ocid>
   ```

## Accessing Your Cluster

After deployment completes, configure `kubectl` to interact with your OKE cluster:

1. **Generate kubeconfig**:
   ```bash
   oci ce cluster create-kubeconfig \
     --cluster-id <cluster-ocid> \
     --file ~/.kube/config \
     --region <region> \
     --profile <oci-profile> \
     --token-version 2.0.0
   ```

2. **Verify cluster access**:
   ```bash
   kubectl cluster-info
   ```
3. **Check cluster status**:
   ```bash
   oci ce cluster get \
     --cluster-id <cluster-ocid> \
     --profile <oci-profile> \
     --query 'data."lifecycle-state"'
   ```

4. **Check nodes** (may take a few minutes to become Ready):
   ```bash
   kubectl get nodes
   ```

5. **Test with a workload**:
   ```bash
   # Deploy test application
   kubectl create deployment nginx-test --image=nginx --replicas=2

   # Check pod status
   kubectl get pods

   # Clean up test
   kubectl delete deployment nginx-test
   ```

## ⚠️ Security Considerations

This example is configured for ease of testing and is NOT prod-ready. The following rules should be restricted in practical use:

1. SSH Access (port 22)
   - Current allows SSH from anywhere
   - `infrastructure/security.pkl`

2. Kubernetes API Endpoint (port 6443)
   - Current allows kubectl access from anywhere
   - `infrastructure/security.pkl`

## Known Issues

- **Destroy may require multiple runs**: NetworkSecurityGroupSecurityRule resources may fail to delete on the first attempt. Run `formae destroy` again if some resources remain. This is being actively addressed.

## Troubleshooting

- kubectl fails to connect: Verify the security list allows ingress on port 6443 from your IP
- Nodes not appearing: Check node pool status in OCI Console; nodes may take 5-10 minutes to provision
- Token auth errors: Ensure OCI CLI is configured correctly and the profile matches
- "Could not find config file": Run `oci setup config` to configure the OCI CLI
