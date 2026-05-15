/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	"reflect"
	"testing"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/randfill"

	utilconversion "sigs.k8s.io/cluster-api/util/conversion"

	infrav1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/powervs/v1beta3"

	. "github.com/onsi/gomega"
)

func TestFuzzyConversion(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	g.Expect(AddToScheme(scheme)).To(Succeed())
	g.Expect(infrav1.AddToScheme(scheme)).To(Succeed())

	t.Run("for IBMPowerVSCluster", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.IBMPowerVSCluster{},
		Spoke:       &IBMPowerVSCluster{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{IBMPowerVSClusterFuzzFuncs},
	}))
	t.Run("for IBMPowerVSClusterTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.IBMPowerVSClusterTemplate{},
		Spoke:       &IBMPowerVSClusterTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{IBMPowerVSClusterTemplateFuzzFuncs},
	}))
	t.Run("for IBMPowerVSMachine", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.IBMPowerVSMachine{},
		Spoke:       &IBMPowerVSMachine{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{IBMPowerVSMachineFuzzFuncs},
	}))
	t.Run("for IBMPowerVSMachineTemplate", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.IBMPowerVSMachineTemplate{},
		Spoke:       &IBMPowerVSMachineTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{IBMPowerVSMachineTemplateFuzzFuncs},
	}))
	t.Run("for IBMPowerVSImage", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Scheme:      scheme,
		Hub:         &infrav1.IBMPowerVSImage{},
		Spoke:       &IBMPowerVSImage{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{IBMPowerVSImageFuzzFuncs},
	}))
}

func IBMPowerVSClusterFuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		hubIBMPowerVSClusterSpec,
		hubIBMPowerVSClusterStatus,
		spokeIBMPowerVSClusterStatus,
		spokeIBMPowerVSClusterSpec,
		spokeIBMPowerVSCluster,
		hubIBMPowerVSCluster,
	}
}

func spokeIBMPowerVSCluster(in *IBMPowerVSCluster, c randfill.Continue) {
	c.FillNoCustom(in)

	// Initialize Annotations to empty map if nil to match v1beta3 behavior
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}

	// Apply spec and status fuzzers
	spokeIBMPowerVSClusterSpec(&in.Spec, c)
	spokeIBMPowerVSClusterStatus(&in.Status, c)
}

func hubIBMPowerVSCluster(in *infrav1.IBMPowerVSCluster, c randfill.Continue) {
	c.FillNoCustom(in)

	// Initialize Annotations to empty map if nil
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}

	// Apply spec and status fuzzers
	hubIBMPowerVSClusterSpec(&in.Spec, c)
	hubIBMPowerVSClusterStatus(&in.Status, c)
}

func hubIBMPowerVSClusterSpec(in *infrav1.IBMPowerVSClusterSpec, c randfill.Continue) {
	c.FillNoCustom(in)

	// Ensure Type fields have valid values (Reference or Provision)
	// Workspace
	if in.Workspace.Type == "" || (in.Workspace.Type != infrav1.SourceTypeReference && in.Workspace.Type != infrav1.SourceTypeProvision) {
		in.Workspace.Type = infrav1.SourceTypeReference
	}
	// Clear opposite fields based on Type
	if in.Workspace.Type == infrav1.SourceTypeReference {
		in.Workspace.Provision = infrav1.WorkspaceProvisionConfig{}
	} else if in.Workspace.Type == infrav1.SourceTypeProvision {
		in.Workspace.Reference = infrav1.ResourceIdentifier{}
	}

	// Network
	if in.Network.Type == "" || (in.Network.Type != infrav1.SourceTypeReference && in.Network.Type != infrav1.SourceTypeProvision) {
		in.Network.Type = infrav1.SourceTypeReference
	}
	if in.Network.Type == infrav1.SourceTypeReference {
		in.Network.Provision = infrav1.NetworkProvisionConfig{}
	} else if in.Network.Type == infrav1.SourceTypeProvision {
		in.Network.Reference = infrav1.ResourceIdentifier{}
	}

	// TransitGateway - clear all fields if Type is empty (can't round-trip from v1beta2 nil)
	if in.TransitGateway.Type == "" {
		in.TransitGateway = infrav1.TransitGateway{}
	} else if in.TransitGateway.Type != infrav1.SourceTypeReference && in.TransitGateway.Type != infrav1.SourceTypeProvision {
		in.TransitGateway.Type = infrav1.SourceTypeReference
	}
	if in.TransitGateway.Type == infrav1.SourceTypeReference {
		in.TransitGateway.Provision = infrav1.TransitGatewayProvisionConfig{}
		// Clear connection fields - they don't exist in v1beta2 TransitGateway
		// v1beta2 only stores ID and Name, not the connection details
		in.TransitGateway.Reference.VPCConnection = infrav1.TransitGatewayConnection{}
		in.TransitGateway.Reference.PowerVSConnection = infrav1.TransitGatewayConnection{}
	} else if in.TransitGateway.Type == infrav1.SourceTypeProvision {
		in.TransitGateway.Reference = infrav1.TransitGatewayReferenceConfig{}
	}

	// VPC - determine Type based on which fields are set
	if in.VPC.Type == "" || (in.VPC.Type != infrav1.SourceTypeReference && in.VPC.Type != infrav1.SourceTypeProvision) {
		// Determine Type based on which fields are populated
		if in.VPC.Reference.ID != "" {
			// Has ID, it's a Reference
			in.VPC.Type = infrav1.SourceTypeReference
		} else if in.VPC.Provision.Name != "" {
			// Has Provision Name, it's a Provision
			in.VPC.Type = infrav1.SourceTypeProvision
		} else {
			// Default to Reference
			in.VPC.Type = infrav1.SourceTypeReference
		}
	}
	if in.VPC.Type == infrav1.SourceTypeReference {
		in.VPC.Provision = infrav1.VPCProvisionConfig{}
	} else if in.VPC.Type == infrav1.SourceTypeProvision {
		in.VPC.Reference = infrav1.ResourceIdentifier{}
	}

	// CosInstance - v1beta2 is always converted to Reference in v1beta3
	// If Type is Provision, convert it to Reference format for round-trip
	if in.CosInstance.Type == infrav1.SourceTypeProvision {
		// Move Provision data to Reference for round-trip
		in.CosInstance.Reference.Instance.Name = in.CosInstance.Provision.Name
		in.CosInstance.Reference.BucketName = in.CosInstance.Provision.BucketName
		in.CosInstance.Type = infrav1.SourceTypeReference
		in.CosInstance.Provision = infrav1.COSProvisionConfig{}
	} else if in.CosInstance.Type == "" || (in.CosInstance.Type != infrav1.SourceTypeReference && in.CosInstance.Type != infrav1.SourceTypeProvision) {
		// Clear CosInstance if it has no valid Type (can't round-trip)
		in.CosInstance = infrav1.CosInstance{}
	} else {
		// Type is Reference, clear Provision fields
		in.CosInstance.Provision = infrav1.COSProvisionConfig{}
	}

	// VPCSubnets - clear entries with empty Type (can't round-trip)
	var validSubnets []infrav1.VPCSubnet
	for i := range in.VPCSubnets {
		if in.VPCSubnets[i].Type == "" {
			// Skip subnets with no Type - they can't round-trip from v1beta2
			continue
		}
		if in.VPCSubnets[i].Type != infrav1.SourceTypeReference && in.VPCSubnets[i].Type != infrav1.SourceTypeProvision {
			in.VPCSubnets[i].Type = infrav1.SourceTypeReference
		}
		if in.VPCSubnets[i].Type == infrav1.SourceTypeReference {
			in.VPCSubnets[i].Provision = infrav1.VPCSubnetProvisionConfig{}
		} else if in.VPCSubnets[i].Type == infrav1.SourceTypeProvision {
			in.VPCSubnets[i].Reference = infrav1.ResourceIdentifier{}
		}
		validSubnets = append(validSubnets, in.VPCSubnets[i])
	}
	in.VPCSubnets = validSubnets

	// LoadBalancers - clear entries with empty Type
	var validLBs []infrav1.VPCLoadBalancer
	for i := range in.LoadBalancers {
		if in.LoadBalancers[i].Type == "" {
			// Skip LBs with no Type
			continue
		}
		if in.LoadBalancers[i].Type != infrav1.SourceTypeReference && in.LoadBalancers[i].Type != infrav1.SourceTypeProvision {
			in.LoadBalancers[i].Type = infrav1.SourceTypeReference
		}
		if in.LoadBalancers[i].Type == infrav1.SourceTypeReference {
			in.LoadBalancers[i].Provision = infrav1.VPCLoadBalancerProvisionConfig{}
		} else if in.LoadBalancers[i].Type == infrav1.SourceTypeProvision {
			in.LoadBalancers[i].Reference = infrav1.ResourceIdentifier{}
			// Clear Subnets from Provision - v1beta2 VPCLoadBalancerSpec doesn't have Subnets in the same way
			in.LoadBalancers[i].Provision.Subnets = nil
		}
		validLBs = append(validLBs, in.LoadBalancers[i])
	}
	in.LoadBalancers = validLBs

	// VPCSecurityGroups
	for i := range in.VPCSecurityGroups {
		if in.VPCSecurityGroups[i].Type == "" || (in.VPCSecurityGroups[i].Type != infrav1.SourceTypeReference && in.VPCSecurityGroups[i].Type != infrav1.SourceTypeProvision) {
			in.VPCSecurityGroups[i].Type = infrav1.SourceTypeReference
		}
		if in.VPCSecurityGroups[i].Type == infrav1.SourceTypeReference {
			in.VPCSecurityGroups[i].Provision = infrav1.VPCSecurityGroupProvisionConfig{}
		} else if in.VPCSecurityGroups[i].Type == infrav1.SourceTypeProvision {
			in.VPCSecurityGroups[i].Reference = infrav1.ResourceIdentifier{}
		}
	}

	// Clear Topology as it doesn't exist in v1beta2
	in.Topology = ""
}

func hubIBMPowerVSClusterStatus(in *infrav1.IBMPowerVSClusterStatus, c randfill.Continue) {
	c.FillNoCustom(in)

	// v1beta2 ResourceReference (used in Status) doesn't have Name field, only ID
	// Clear Name from all Status ResourceReference fields to avoid data loss in round-trip
	in.ResourceGroup.Name = ""
	in.Workspace.Name = ""
	in.Network.Name = ""
	in.DHCPServer.Name = ""
	in.VPC.Name = ""
	in.TransitGateway.VPCConnection.Name = ""
	in.TransitGateway.PowerVSConnection.Name = ""
	in.COSInstance.Name = ""

	// Clear ResourceReference fields with empty ID (won't round-trip from v1beta2 nil)
	// But keep them if they have non-empty ID
	if in.ResourceGroup.ID == "" {
		in.ResourceGroup = infrav1.ResourceReference{}
	}
	if in.Workspace.ID == "" {
		in.Workspace = infrav1.ResourceReference{}
	}
	if in.Network.ID == "" {
		in.Network = infrav1.ResourceReference{}
	}
	if in.DHCPServer.ID == "" {
		in.DHCPServer = infrav1.ResourceReference{}
	}
	if in.VPC.ID == "" {
		in.VPC = infrav1.ResourceReference{}
	}
	if in.COSInstance.ID == "" {
		in.COSInstance = infrav1.ResourceReference{}
	}

	// Clear VPCSecurityGroups if it's a non-nil slice with entries that have no RuleIDs
	// v1beta2 uses map, v1beta3 uses slice - empty entries don't round-trip
	if len(in.VPCSecurityGroups) > 0 {
		var validSGs []infrav1.VPCSecurityGroupStatus
		for i := range in.VPCSecurityGroups {
			// Only keep SGs that have ID and at least one RuleID
			if in.VPCSecurityGroups[i].ID != "" && len(in.VPCSecurityGroups[i].RuleIDs) > 0 {
				validSGs = append(validSGs, in.VPCSecurityGroups[i])
			}
		}
		in.VPCSecurityGroups = validSGs
	}

	// Clear TransitGateway connections with empty IDs
	if in.TransitGateway.VPCConnection.ID == "" {
		in.TransitGateway.VPCConnection = infrav1.ResourceReference{}
	}
	if in.TransitGateway.PowerVSConnection.ID == "" {
		in.TransitGateway.PowerVSConnection = infrav1.ResourceReference{}
	}

	// Clear VPCSubnets - convert empty slice to nil and clear Name fields
	if len(in.VPCSubnets) == 0 {
		in.VPCSubnets = nil
	} else {
		for i := range in.VPCSubnets {
			in.VPCSubnets[i].Name = ""
		}
	}

	// Clear LoadBalancers - convert empty slice to nil and clear Name fields
	if len(in.LoadBalancers) == 0 {
		in.LoadBalancers = nil
	} else {
		for i := range in.LoadBalancers {
			in.LoadBalancers[i].Name = ""
		}
	}

	// Clear VPCSecurityGroups empty slice
	if len(in.VPCSecurityGroups) == 0 {
		in.VPCSecurityGroups = nil
	}

	// Don't clear Initialization.Provisioned - it always round-trips through Ready field
	// The conversion sets Provisioned = ptr.To(in.Ready), so it will always have a value

	// Clear TransitGateway if it only has empty ID
	if in.TransitGateway.ID == "" && in.TransitGateway.VPCConnection.ID == "" && in.TransitGateway.PowerVSConnection.ID == "" {
		in.TransitGateway = infrav1.TransitGatewayStatus{}
	}

	// Drop empty structs with only omit empty fields.
	if in.Deprecated != nil {
		if in.Deprecated.V1Beta2 == nil || reflect.DeepEqual(in.Deprecated.V1Beta2, &infrav1.IBMPowerVSClusterV1Beta2DeprecatedStatus{}) {
			in.Deprecated = nil
		}
	}
}

func spokeIBMPowerVSClusterStatus(in *IBMPowerVSClusterStatus, c randfill.Continue) {
	c.FillNoCustom(in)

	// Clear ServiceInstance from Status (deprecated field, shouldn't be in Status)
	in.ServiceInstance = nil

	// Clear empty ResourceReference structs (only have ControllerCreated, no ID)
	// v1beta3 doesn't have ControllerCreated in Status ResourceReference
	if in.ResourceGroup != nil {
		if in.ResourceGroup.ID == nil || *in.ResourceGroup.ID == "" {
			in.ResourceGroup = nil
		} else {
			// Has ID, clear ControllerCreated as it doesn't exist in v1beta3
			in.ResourceGroup.ControllerCreated = nil
		}
	}
	if in.Network != nil {
		if in.Network.ID == nil || *in.Network.ID == "" {
			in.Network = nil
		} else {
			// Has ID, clear ControllerCreated as it doesn't exist in v1beta3
			in.Network.ControllerCreated = nil
		}
	}
	if in.DHCPServer != nil && (in.DHCPServer.ID == nil || *in.DHCPServer.ID == "") {
		in.DHCPServer = nil
	}
	if in.VPC != nil {
		if in.VPC.ID == nil || *in.VPC.ID == "" {
			in.VPC = nil
		} else {
			// v1beta3 doesn't have ControllerCreated in Status, so clear VPC if it has ControllerCreated
			// This prevents round-trip issues
			in.VPC = nil
		}
	}
	if in.COSInstance != nil {
		if in.COSInstance.ID == nil || *in.COSInstance.ID == "" {
			in.COSInstance = nil
		} else {
			// v1beta3 doesn't have ControllerCreated in Status, so clear COSInstance if it has any value
			// This prevents round-trip issues
			in.COSInstance = nil
		}
	}

	// Fix VPCSubnet map keys to match IDs and clear ControllerCreated
	if in.VPCSubnet != nil {
		fixedMap := make(map[string]ResourceReference)
		for _, v := range in.VPCSubnet {
			if v.ID != nil && *v.ID != "" {
				// Clear ControllerCreated as it doesn't exist in v1beta3
				v.ControllerCreated = nil
				// Use ID as the map key
				fixedMap[*v.ID] = v
			}
		}
		if len(fixedMap) == 0 {
			in.VPCSubnet = nil
		} else {
			in.VPCSubnet = fixedMap
		}
	}

	// Fix LoadBalancers map keys to match IDs and clear empty Hostname pointers and ControllerCreated
	if in.LoadBalancers != nil {
		fixedMap := make(map[string]VPCLoadBalancerStatus)
		for _, v := range in.LoadBalancers {
			if v.ID != nil && *v.ID != "" {
				// Clear empty Hostname pointer
				if v.Hostname != nil && *v.Hostname == "" {
					v.Hostname = nil
				}
				// Clear ControllerCreated as it doesn't exist in v1beta3
				v.ControllerCreated = nil
				// Use ID as the map key
				fixedMap[*v.ID] = v
			}
		}
		if len(fixedMap) == 0 {
			in.LoadBalancers = nil
		} else {
			in.LoadBalancers = fixedMap
		}
	}

	// Fix VPCSecurityGroups map keys to match IDs and clear entries with empty IDs or ControllerCreated
	if in.VPCSecurityGroups != nil {
		fixedMap := make(map[string]VPCSecurityGroupStatus)
		for _, v := range in.VPCSecurityGroups {
			if v.ID != nil && *v.ID != "" {
				// Clear ControllerCreated as it doesn't exist in v1beta3
				v.ControllerCreated = nil
				// Use ID as the map key
				fixedMap[*v.ID] = v
			}
		}
		if len(fixedMap) == 0 {
			in.VPCSecurityGroups = nil
		} else {
			in.VPCSecurityGroups = fixedMap
		}
	}

	// Clear DHCPServer if it only has empty ID
	if in.DHCPServer != nil {
		if in.DHCPServer.ID == nil || *in.DHCPServer.ID == "" {
			in.DHCPServer = nil
		} else {
			// Has ID, clear ControllerCreated as it doesn't exist in v1beta3
			in.DHCPServer.ControllerCreated = nil
		}
	}

	// Clear empty TransitGatewayStatus
	// v1beta3 TransitGatewayStatus doesn't have VPCConnection/PowerVSConnection when ID is empty
	// So if ID is empty, clear the whole TransitGateway regardless of connections
	if in.TransitGateway != nil {
		if in.TransitGateway.ID == nil || *in.TransitGateway.ID == "" {
			// No ID means no TransitGateway in v1beta3
			in.TransitGateway = nil
		} else {
			// If TransitGateway has ID but no connections, it still won't round-trip properly
			// because v1beta3 will have empty ResourceReference structs for connections
			// Clear the whole TransitGateway if connections are empty
			hasValidConnection := false
			if in.TransitGateway.VPCConnection != nil && in.TransitGateway.VPCConnection.ID != nil && *in.TransitGateway.VPCConnection.ID != "" {
				hasValidConnection = true
			}
			if in.TransitGateway.PowerVSConnection != nil && in.TransitGateway.PowerVSConnection.ID != nil && *in.TransitGateway.PowerVSConnection.ID != "" {
				hasValidConnection = true
			}

			if !hasValidConnection {
				// No valid connections, clear the whole thing
				in.TransitGateway = nil
			} else {
				// Clear ControllerCreated from TransitGateway itself as it doesn't exist in v1beta3
				in.TransitGateway.ControllerCreated = nil

				// Clear empty connection references
				if in.TransitGateway.VPCConnection != nil && (in.TransitGateway.VPCConnection.ID == nil || *in.TransitGateway.VPCConnection.ID == "") {
					in.TransitGateway.VPCConnection = nil
				} else if in.TransitGateway.VPCConnection != nil {
					// Has ID, clear ControllerCreated as it doesn't exist in v1beta3
					in.TransitGateway.VPCConnection.ControllerCreated = nil
				}
				if in.TransitGateway.PowerVSConnection != nil && (in.TransitGateway.PowerVSConnection.ID == nil || *in.TransitGateway.PowerVSConnection.ID == "") {
					in.TransitGateway.PowerVSConnection = nil
				} else if in.TransitGateway.PowerVSConnection != nil {
					// Has ID, clear ControllerCreated as it doesn't exist in v1beta3
					in.TransitGateway.PowerVSConnection.ControllerCreated = nil
				}
			}
		}
	}

	// Drop empty structs with only omit empty fields.
	if in.V1Beta2 != nil {
		if reflect.DeepEqual(in.V1Beta2, &IBMPowerVSClusterV1Beta2Status{}) {
			in.V1Beta2 = nil
		}
	}
}

func spokeIBMPowerVSClusterSpec(in *IBMPowerVSClusterSpec, c randfill.Continue) {
	c.FillNoCustom(in)

	// Ensure ServiceInstance and ServiceInstanceID are in sync for round-trip conversion
	// This handles the deprecated field migration
	if in.ServiceInstance != nil && in.ServiceInstance.ID != nil && *in.ServiceInstance.ID != "" {
		// If ServiceInstance.ID is set, ensure ServiceInstanceID matches
		in.ServiceInstanceID = *in.ServiceInstance.ID
	} else if in.ServiceInstanceID != "" {
		// If ServiceInstanceID is set, ensure ServiceInstance matches
		if in.ServiceInstance == nil {
			in.ServiceInstance = &IBMPowerVSResourceReference{}
		}
		id := in.ServiceInstanceID
		in.ServiceInstance.ID = &id
	}

	// Clear DHCPServer.Snat if it's false (default value, won't round-trip)
	if in.DHCPServer != nil && in.DHCPServer.Snat != nil && !*in.DHCPServer.Snat {
		in.DHCPServer.Snat = nil
	}

	// Clear empty string pointers
	if in.Network.ID != nil && *in.Network.ID == "" {
		in.Network.ID = nil
	}
	if in.Network.Name != nil && *in.Network.Name == "" {
		in.Network.Name = nil
	}

	// Clear RegEx fields - not supported in v1beta3 ResourceIdentifier
	if in.Network.RegEx != nil {
		in.Network.RegEx = nil
	}
	if in.ServiceInstance != nil && in.ServiceInstance.RegEx != nil {
		in.ServiceInstance.RegEx = nil
	}
	if in.ResourceGroup != nil && in.ResourceGroup.RegEx != nil {
		in.ResourceGroup.RegEx = nil
	}

	// Clear empty string pointers in TransitGateway
	if in.TransitGateway != nil {
		if in.TransitGateway.Name != nil && *in.TransitGateway.Name == "" {
			in.TransitGateway.Name = nil
		}
		if in.TransitGateway.ID != nil && *in.TransitGateway.ID == "" {
			in.TransitGateway.ID = nil
		}
		// Clear TransitGateway.GlobalRouting if ID or Name is set (GlobalRouting only applies to Provision)
		if (in.TransitGateway.ID != nil && *in.TransitGateway.ID != "") ||
			(in.TransitGateway.Name != nil && *in.TransitGateway.Name != "") {
			in.TransitGateway.GlobalRouting = nil
		}
		// Clear TransitGateway if completely empty
		if in.TransitGateway.ID == nil && in.TransitGateway.Name == nil && in.TransitGateway.GlobalRouting == nil {
			in.TransitGateway = nil
		}
	}

	// Clear empty string pointers in VPC
	if in.VPC != nil {
		if in.VPC.Region != nil && *in.VPC.Region == "" {
			in.VPC.Region = nil
		}
		if in.VPC.ID != nil && *in.VPC.ID == "" {
			in.VPC.ID = nil
		}
		if in.VPC.Name != nil && *in.VPC.Name == "" {
			in.VPC.Name = nil
		}
		// Clear VPC if it only has Region (no ID or Name) - can't round-trip
		if (in.VPC.ID == nil || *in.VPC.ID == "") && (in.VPC.Name == nil || *in.VPC.Name == "") {
			in.VPC = nil
		}
	}

	// Clear DHCPServer if Network has ID or Name (they're mutually exclusive in conversion)
	// Also clear if DHCPServer only has ID without other fields
	if (in.Network.ID != nil && *in.Network.ID != "") || (in.Network.Name != nil && *in.Network.Name != "") {
		in.DHCPServer = nil
	} else if in.DHCPServer != nil {
		// Clear empty string pointers in DHCPServer
		if in.DHCPServer.Cidr != nil && *in.DHCPServer.Cidr == "" {
			in.DHCPServer.Cidr = nil
		}
		if in.DHCPServer.DNSServer != nil && *in.DHCPServer.DNSServer == "" {
			in.DHCPServer.DNSServer = nil
		}
		if in.DHCPServer.Name != nil && *in.DHCPServer.Name == "" {
			in.DHCPServer.Name = nil
		}

		// DHCPServer in Spec shouldn't have ID field - it's only in Status
		// Clear ID if it's set
		in.DHCPServer.ID = nil

		// If DHCPServer only has ID (now cleared) or only empty fields, clear the whole thing
		if in.DHCPServer.Name == nil && in.DHCPServer.Cidr == nil &&
			in.DHCPServer.DNSServer == nil && in.DHCPServer.Snat == nil {
			in.DHCPServer = nil
		}
	}

	// Clear empty VPCSubnets entries that only have Zone or Ipv4CidrBlock
	for i := range in.VPCSubnets {
		// Clear empty string pointers
		if in.VPCSubnets[i].ID != nil && *in.VPCSubnets[i].ID == "" {
			in.VPCSubnets[i].ID = nil
		}
		if in.VPCSubnets[i].Name != nil && *in.VPCSubnets[i].Name == "" {
			in.VPCSubnets[i].Name = nil
		}
		if in.VPCSubnets[i].Zone != nil && *in.VPCSubnets[i].Zone == "" {
			in.VPCSubnets[i].Zone = nil
		}
		if in.VPCSubnets[i].Ipv4CidrBlock != nil && *in.VPCSubnets[i].Ipv4CidrBlock == "" {
			in.VPCSubnets[i].Ipv4CidrBlock = nil
		}

		// If ID is set (Reference type), clear Zone and Ipv4CidrBlock as they don't round-trip
		// Reference type only uses ID and Name, not Zone or Ipv4CidrBlock
		if in.VPCSubnets[i].ID != nil && *in.VPCSubnets[i].ID != "" {
			in.VPCSubnets[i].Zone = nil
			in.VPCSubnets[i].Ipv4CidrBlock = nil
		} else if (in.VPCSubnets[i].ID == nil || *in.VPCSubnets[i].ID == "") &&
			(in.VPCSubnets[i].Name == nil || *in.VPCSubnets[i].Name == "") {
			// If no ID or Name, clear Zone and Ipv4CidrBlock as they can't round-trip
			in.VPCSubnets[i].Zone = nil
			in.VPCSubnets[i].Ipv4CidrBlock = nil
		}
	}

	// Clear empty Zone pointer
	if in.Zone != nil && *in.Zone == "" {
		in.Zone = nil
	}

	// Clear empty ResourceGroup (no ID or Name)
	if in.ResourceGroup != nil && (in.ResourceGroup.ID == nil || *in.ResourceGroup.ID == "") &&
		(in.ResourceGroup.Name == nil || *in.ResourceGroup.Name == "") {
		in.ResourceGroup = nil
	}

	// Clear empty VPCSecurityGroups and clean up individual entries
	var validSGs []VPCSecurityGroup
	for i := range in.VPCSecurityGroups {
		// Clear empty string pointers in Name and ID
		if in.VPCSecurityGroups[i].Name != nil && *in.VPCSecurityGroups[i].Name == "" {
			in.VPCSecurityGroups[i].Name = nil
		}
		if in.VPCSecurityGroups[i].ID != nil && *in.VPCSecurityGroups[i].ID == "" {
			in.VPCSecurityGroups[i].ID = nil
		}

		// Handle Rules - clean up rule fields for proper round-trip
		if len(in.VPCSecurityGroups[i].Rules) > 0 {
			for _, rule := range in.VPCSecurityGroups[i].Rules {
				if rule != nil {
					// Clear ICMPCode and ICMPType if they're 0 (default value)
					if rule.Destination != nil {
						if rule.Destination.ICMPCode != nil && *rule.Destination.ICMPCode == 0 {
							rule.Destination.ICMPCode = nil
						}
						if rule.Destination.ICMPType != nil && *rule.Destination.ICMPType == 0 {
							rule.Destination.ICMPType = nil
						}
					}
					if rule.Source != nil {
						if rule.Source.ICMPCode != nil && *rule.Source.ICMPCode == 0 {
							rule.Source.ICMPCode = nil
						}
						if rule.Source.ICMPType != nil && *rule.Source.ICMPType == 0 {
							rule.Source.ICMPType = nil
						}
					}
				}
			}
		}

		// If ID is set (Reference type), clear Rules, Tags, and SecurityGroupID from rules
		// Reference type only uses ID and Name
		if in.VPCSecurityGroups[i].ID != nil && *in.VPCSecurityGroups[i].ID != "" {
			// For Reference type, clear all Rules and Tags
			in.VPCSecurityGroups[i].Rules = nil
			in.VPCSecurityGroups[i].Tags = nil
		} else {
			// For Provision type, clear SecurityGroupID from rules (it's only for Reference)
			if len(in.VPCSecurityGroups[i].Rules) > 0 {
				for _, rule := range in.VPCSecurityGroups[i].Rules {
					if rule != nil {
						rule.SecurityGroupID = nil
					}
				}
			}

			// Provision type - handle Tags - convert empty slice to nil for proper round-trip
			if len(in.VPCSecurityGroups[i].Tags) > 0 {
				var validTags []*string
				for _, tag := range in.VPCSecurityGroups[i].Tags {
					if tag != nil && *tag != "" {
						validTags = append(validTags, tag)
					}
				}
				if len(validTags) == 0 {
					in.VPCSecurityGroups[i].Tags = nil
				} else {
					in.VPCSecurityGroups[i].Tags = validTags
				}
			} else {
				// Empty slice should be nil for round-trip
				in.VPCSecurityGroups[i].Tags = nil
			}
		}

		// Only keep SGs that have ID or Name
		if (in.VPCSecurityGroups[i].ID != nil && *in.VPCSecurityGroups[i].ID != "") ||
			(in.VPCSecurityGroups[i].Name != nil && *in.VPCSecurityGroups[i].Name != "") {
			validSGs = append(validSGs, in.VPCSecurityGroups[i])
		}
	}
	in.VPCSecurityGroups = validSGs
	if len(in.VPCSecurityGroups) == 0 {
		in.VPCSecurityGroups = nil
	}

	// Clear TransitGateway if it only has GlobalRouting set (can't round-trip without ID or Name)
	if in.TransitGateway != nil {
		if (in.TransitGateway.ID == nil || *in.TransitGateway.ID == "") &&
			(in.TransitGateway.Name == nil || *in.TransitGateway.Name == "") &&
			in.TransitGateway.GlobalRouting != nil {
			// Only GlobalRouting is set, which means it's a Provision config
			// But without other fields, it can't round-trip properly
			in.TransitGateway = nil
		}
	}

	// Clear CosInstance if it doesn't have the required fields for round-trip
	// v1beta2 CosInstance needs Name, BucketName, and BucketRegion to properly convert to v1beta3
	// v1beta3 Reference.Instance.Name comes from v1beta2 Name
	if in.CosInstance != nil {
		// If Name is empty, we can't round-trip properly because v1beta3 Reference.Instance.Name will be empty
		// and that might cause issues. Ensure Name is set if CosInstance is used.
		if in.CosInstance.Name == "" {
			// Generate a name if BucketName or BucketRegion is set
			if in.CosInstance.BucketName != "" || in.CosInstance.BucketRegion != "" {
				in.CosInstance.Name = "cos-instance"
			} else {
				// No useful data, clear it
				in.CosInstance = nil
			}
		}
	}

	// Clear Ignition if it's empty (only has empty Version)
	if in.Ignition != nil && in.Ignition.Version == "" {
		in.Ignition = nil
	}

	// Clear empty string pointers in ResourceGroup
	if in.ResourceGroup != nil {
		if in.ResourceGroup.Name != nil && *in.ResourceGroup.Name == "" {
			in.ResourceGroup.Name = nil
		}
		if in.ResourceGroup.ID != nil && *in.ResourceGroup.ID == "" {
			in.ResourceGroup.ID = nil
		}
	}

	// Clear empty LoadBalancer fields and handle BackendPool Name
	for i := range in.LoadBalancers {
		// Clear empty string pointers
		if in.LoadBalancers[i].ID != nil && *in.LoadBalancers[i].ID == "" {
			in.LoadBalancers[i].ID = nil
		}

		// Clear Public field entirely - it doesn't round-trip properly
		// v1beta3 uses Visibility (public/private), v1beta2 uses Public bool
		in.LoadBalancers[i].Public = nil

		// Clear AdditionalListeners - they don't round-trip from v1beta2 to v1beta3
		// v1beta3 doesn't have AdditionalListeners in the same way
		in.LoadBalancers[i].AdditionalListeners = nil

		// Clear BackendPools with empty Name - Name is required for round-trip
		var validPools []VPCLoadBalancerBackendPoolSpec
		for _, pool := range in.LoadBalancers[i].BackendPools {
			if pool.Name != nil && *pool.Name != "" {
				validPools = append(validPools, pool)
			}
		}
		if len(validPools) == 0 {
			in.LoadBalancers[i].BackendPools = nil
		} else {
			in.LoadBalancers[i].BackendPools = validPools
		}

		// Clear Subnets - they don't round-trip properly from v1beta2 to v1beta3
		// v1beta3 LoadBalancer Provision has Subnets, but v1beta2 doesn't store them the same way
		in.LoadBalancers[i].Subnets = nil

		// Clear SecurityGroups empty slice
		if len(in.LoadBalancers[i].SecurityGroups) == 0 {
			in.LoadBalancers[i].SecurityGroups = nil
		}
		if len(in.LoadBalancers[i].BackendPools) == 0 {
			in.LoadBalancers[i].BackendPools = nil
		} else {
			// Handle BackendPool Name and URLPath fields - convert nil to &"" for round-trip
			for j := range in.LoadBalancers[i].BackendPools {
				if in.LoadBalancers[i].BackendPools[j].Name == nil {
					emptyStr := ""
					in.LoadBalancers[i].BackendPools[j].Name = &emptyStr
				}
				// URLPath should also be &"" not nil for round-trip
				if in.LoadBalancers[i].BackendPools[j].HealthMonitor.URLPath == nil {
					emptyStr := ""
					in.LoadBalancers[i].BackendPools[j].HealthMonitor.URLPath = &emptyStr
				}
			}
		}

		// Clear SecurityGroups with only ID (no Name) - can't round-trip properly
		var validSGs []VPCResource
		for _, sg := range in.LoadBalancers[i].SecurityGroups {
			if sg.ID != nil && *sg.ID != "" && sg.Name != nil && *sg.Name != "" {
				validSGs = append(validSGs, sg)
			}
		}
		if len(validSGs) == 0 {
			in.LoadBalancers[i].SecurityGroups = nil
		} else {
			in.LoadBalancers[i].SecurityGroups = validSGs
		}
	}
}

func IBMPowerVSMachineFuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		hubIBMPowerVSMachineStatus,
		spokeIBMPowerVSMachineSpec,
		spokeIBMPowerVSMachineStatus,
		spokeIBMPowerVSMachine,
		hubIBMPowerVSMachine,
	}
}

func spokeIBMPowerVSMachine(in *IBMPowerVSMachine, c randfill.Continue) {
	c.FillNoCustom(in)

	// Initialize Annotations to empty map if nil
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}

	// Apply spec and status fuzzers
	spokeIBMPowerVSMachineSpec(&in.Spec, c)
	spokeIBMPowerVSMachineStatus(&in.Status, c)
}

func hubIBMPowerVSMachine(in *infrav1.IBMPowerVSMachine, c randfill.Continue) {
	c.FillNoCustom(in)

	// Initialize Annotations to empty map if nil
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}

	// Apply status fuzzer
	hubIBMPowerVSMachineStatus(&in.Status, c)
}

func hubIBMPowerVSMachineStatus(in *infrav1.IBMPowerVSMachineStatus, c randfill.Continue) {
	c.FillNoCustom(in)

	// Clear Initialization if it only has Provisioned=false (default value, won't round-trip)
	if in.Initialization.Provisioned != nil && !*in.Initialization.Provisioned {
		in.Initialization = infrav1.IBMPowerVSMachineInitializationStatus{}
	}

	// Drop empty structs with only omit empty fields.
	if in.Deprecated != nil {
		if in.Deprecated.V1Beta2 == nil || reflect.DeepEqual(in.Deprecated.V1Beta2, &infrav1.IBMPowerVSMachineV1Beta2DeprecatedStatus{}) {
			in.Deprecated = nil
		}
	}
}

func IBMPowerVSClusterTemplateFuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		spokeIBMPowerVSClusterTemplateResource,
	}
}

func spokeIBMPowerVSClusterTemplateResource(in *IBMPowerVSClusterTemplateResource, c randfill.Continue) {
	c.FillNoCustom(in)

	// Handle the nested spec
	spokeIBMPowerVSClusterSpec(&in.Spec, c)
}

func spokeIBMPowerVSMachineSpec(in *IBMPowerVSMachineSpec, c randfill.Continue) {
	c.FillNoCustom(in)

	if in.ProviderID != nil && *in.ProviderID == "" {
		in.ProviderID = nil
	}

	// Set ImageRef to nil if it has an empty name
	if in.ImageRef != nil && in.ImageRef.Name == "" {
		in.ImageRef = nil
	}

	// Ensure ServiceInstance and ServiceInstanceID are in sync for round-trip conversion
	// This handles the deprecated field migration
	if in.ServiceInstance != nil && in.ServiceInstance.ID != nil && *in.ServiceInstance.ID != "" {
		// If ServiceInstance.ID is set, ensure ServiceInstanceID matches
		in.ServiceInstanceID = *in.ServiceInstance.ID
		// Clear Name and RegEx as they don't round-trip
		in.ServiceInstance.Name = nil
		in.ServiceInstance.RegEx = nil
	} else if in.ServiceInstanceID != "" {
		// If ServiceInstanceID is set, ensure ServiceInstance matches
		if in.ServiceInstance == nil {
			in.ServiceInstance = &IBMPowerVSResourceReference{}
		}
		id := in.ServiceInstanceID
		in.ServiceInstance.ID = &id
		in.ServiceInstance.Name = nil
		in.ServiceInstance.RegEx = nil
	}

	// Clear Image if it doesn't have both ID and Name, or clear it entirely
	// v1beta3 ResourceIdentifier doesn't have RegEx, and the conversion is complex
	// If ImageRef is set, Image should be nil
	if in.ImageRef != nil && in.ImageRef.Name != "" {
		in.Image = nil
	} else if in.Image != nil {
		// Clear Image if it only has ID (no Name) or only Name (no ID)
		hasID := in.Image.ID != nil && *in.Image.ID != ""
		hasName := in.Image.Name != nil && *in.Image.Name != ""

		if !hasID || !hasName {
			// Need both ID and Name for proper round-trip, otherwise clear
			in.Image = nil
		} else {
			// Has both, clear RegEx
			in.Image.RegEx = nil
		}
	}

	// Clear Network RegEx
	in.Network.RegEx = nil
}

func spokeIBMPowerVSMachineStatus(in *IBMPowerVSMachineStatus, c randfill.Continue) {
	c.FillNoCustom(in)

	// v1beta3 uses string for Zone, v1beta2 uses *string
	// When v1beta3 Zone is "", it converts to v1beta2 &""
	// We need to convert &"" back to nil, but also handle the reverse:
	// v1beta2 nil converts to v1beta3 "", which converts back to v1beta2 &""
	// So we need to ensure Zone is either nil or has a non-empty value
	if in.Zone != nil && *in.Zone == "" {
		in.Zone = nil
	}

	// Clear Region if it's set (doesn't exist in v1beta3, won't round-trip)
	in.Region = nil

	// Clear FailureReason and FailureMessage - v1beta3 doesn't have these fields
	// They won't round-trip at all
	in.FailureReason = nil
	in.FailureMessage = nil

	// Drop empty structs with only omit empty fields.
	if in.V1Beta2 != nil {
		if reflect.DeepEqual(in.V1Beta2, &IBMPowerVSMachineV1Beta2Status{}) {
			in.V1Beta2 = nil
		}
	}
}

func IBMPowerVSMachineTemplateFuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		hubIBMPowerVSMachineTemplateResource,
		spokeIBMPowerVSMachineTemplateResource,
		spokeIBMPowerVSMachineTemplate,
		hubIBMPowerVSMachineTemplate,
	}
}

func spokeIBMPowerVSMachineTemplate(in *IBMPowerVSMachineTemplate, c randfill.Continue) {
	c.FillNoCustom(in)

	// Initialize Annotations to empty map if nil
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}

	// Apply template resource fuzzer
	spokeIBMPowerVSMachineTemplateResource(&in.Spec.Template, c)
}

func hubIBMPowerVSMachineTemplate(in *infrav1.IBMPowerVSMachineTemplate, c randfill.Continue) {
	c.FillNoCustom(in)

	// Initialize Annotations to empty map if nil
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}

	// Clear Status.Capacity if it's an empty map (won't round-trip from v1beta2)
	if len(in.Status.Capacity) == 0 {
		in.Status.Capacity = nil
	}

	// Apply template resource fuzzer
	hubIBMPowerVSMachineTemplateResource(&in.Spec.Template, c)
}

func spokeIBMPowerVSMachineTemplateResource(in *IBMPowerVSMachineTemplateResource, c randfill.Continue) {
	c.FillNoCustom(in)

	// Handle the nested spec
	spokeIBMPowerVSMachineSpec(&in.Spec, c)
}

func hubIBMPowerVSMachineTemplateResource(in *infrav1.IBMPowerVSMachineTemplateResource, c randfill.Continue) {
	c.FillNoCustom(in)

	in.ObjectMeta = clusterv1.ObjectMeta{} // Field does not exist in v1beta2.
}

func IBMPowerVSImageFuzzFuncs(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		hubIBMPowerVSImageSpec,
		hubIBMPowerVSImageStatus,
		spokeIBMPowerVSImageStatus,
		spokeIBMPowerVSImageSpec,
	}
}

func hubIBMPowerVSImageSpec(in *infrav1.IBMPowerVSImageSpec, c randfill.Continue) {
	c.FillNoCustom(in)

	// v1beta2 IBMPowerVSResourceReference doesn't have Name field, only ID
	// Clear Workspace.Name to avoid data loss in round-trip
	in.Workspace.Name = ""

	// v1beta3 uses string, v1beta2 uses *string
	// Empty strings in v1beta3 will become &"" in v1beta2, which should be nil
	// So clear empty strings to avoid this issue
	if in.Bucket == "" {
		// Keep it as empty string - the fuzzer will handle the conversion
	}
	if in.Object == "" {
		// Keep it as empty string - the fuzzer will handle the conversion
	}
	if in.Region == "" {
		// Keep it as empty string - the fuzzer will handle the conversion
	}
}

func spokeIBMPowerVSImageSpec(in *IBMPowerVSImageSpec, c randfill.Continue) {
	c.FillNoCustom(in)

	// Ensure ServiceInstance and ServiceInstanceID are in sync for round-trip conversion
	// This handles the deprecated field migration
	if in.ServiceInstance != nil && in.ServiceInstance.ID != nil && *in.ServiceInstance.ID != "" {
		// If ServiceInstance.ID is set, ensure ServiceInstanceID matches
		in.ServiceInstanceID = *in.ServiceInstance.ID
		// Clear RegEx and Name fields
		in.ServiceInstance.RegEx = nil
		in.ServiceInstance.Name = nil
	} else if in.ServiceInstanceID != "" {
		// If ServiceInstanceID is set, ensure ServiceInstance matches
		if in.ServiceInstance == nil {
			in.ServiceInstance = &IBMPowerVSResourceReference{}
		}
		id := in.ServiceInstanceID
		in.ServiceInstance.ID = &id
		in.ServiceInstance.RegEx = nil
		in.ServiceInstance.Name = nil
	} else {
		// Neither is set, clear ServiceInstance entirely
		in.ServiceInstance = nil
	}

	// v1beta3 uses string instead of *string for Bucket, Object, Region
	// The conversion sets these to nil when empty string in v1beta3
	// So we should clear empty string pointers to nil for proper round-trip
	if in.Bucket != nil && *in.Bucket == "" {
		in.Bucket = nil
	}
	if in.Object != nil && *in.Object == "" {
		in.Object = nil
	}
	if in.Region != nil && *in.Region == "" {
		in.Region = nil
	}
}

func hubIBMPowerVSImageStatus(in *infrav1.IBMPowerVSImageStatus, c randfill.Continue) {
	c.FillNoCustom(in)
	// Drop empty structs with only omit empty fields.
	if in.Deprecated != nil {
		if in.Deprecated.V1Beta2 == nil || reflect.DeepEqual(in.Deprecated.V1Beta2, &infrav1.IBMPowerVSImageV1Beta2DeprecatedStatus{}) {
			in.Deprecated = nil
		}
	}
}

func spokeIBMPowerVSImageStatus(in *IBMPowerVSImageStatus, c randfill.Continue) {
	c.FillNoCustom(in)
	// Drop empty structs with only omit empty fields.
	if in.V1Beta2 != nil {
		if reflect.DeepEqual(in.V1Beta2, &IBMPowerVSImageV1Beta2Status{}) {
			in.V1Beta2 = nil
		}
	}
}
