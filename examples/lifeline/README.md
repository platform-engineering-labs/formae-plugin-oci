# OCI Lifeline Example

This example demonstrates a basic OCI infrastructure with VCN, subnets, and network security groups.

## Files

- `basic_infrastructure.pkl` - Core infrastructure (VCN, subnets, gateways, route tables)
- `security_group_resources.pkl` - Network security groups and rules
- `cross_cutting_change.pkl` - Patch document for updating multiple resources
- `micro_change.pkl` - Patch document for small changes
- `vars.pkl` - Configuration variables
- `target.pkl` - OCI target configuration

## Usage

1. Apply the basic infrastructure:
   ```bash
   formae apply basic_infrastructure.pkl
   ```

2. Apply patches as needed:
   ```bash
   formae apply --mode patch cross_cutting_change.pkl
   formae apply --mode patch micro_change.pkl
   ```

3. Destroy when done:
   ```bash
   formae destroy --query
   ```

## Known Issues

- **Destroy may require multiple runs**: NetworkSecurityGroupSecurityRule resources may fail to delete on the first attempt. Run `formae destroy` again if some resources remain. This is being actively addressed.
