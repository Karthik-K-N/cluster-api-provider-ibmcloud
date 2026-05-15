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

package v1beta3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

const (
	// IBMPowerVSClusterFinalizer allows IBMPowerVSClusterReconciler to clean up resources associated with IBMPowerVSCluster before
	// removing it from the apiserver.
	IBMPowerVSClusterFinalizer = "ibmpowervscluster.infrastructure.cluster.x-k8s.io"
)

// SourceType defines the provisioning strategy for a resource.
type SourceType string

const (
	// SourceTypeReference indicates the controller should use an existing resource.
	SourceTypeReference SourceType = "Reference"

	// SourceTypeProvision indicates the controller should create a new resource.
	SourceTypeProvision SourceType = "Provision"
)

// ClusterTopology defines the external access architecture of the cluster.
type ClusterTopology string

const (
	// PowerVSVIPTopology uses a pure PowerVS network and Virtual IP for access.
	PowerVSVIPTopology ClusterTopology = "VIP"

	// PowerVSLoadBalancerTopology integrates the PowerVS workspace with an IBM Cloud VPC and LoadBalancer.
	PowerVSLoadBalancerTopology ClusterTopology = "LoadBalancer"
)

// DHCPSnatPolicy defines the SNAT policy for the DHCP service.
type DHCPSnatPolicy string

const (
	// DHCPSnatPolicyEnabled indicates that SNAT is enabled for the DHCP service.
	DHCPSnatPolicyEnabled DHCPSnatPolicy = "Enabled"

	// DHCPSnatPolicyDisabled indicates that SNAT is disabled for the DHCP service.
	DHCPSnatPolicyDisabled DHCPSnatPolicy = "Disabled"
)

type TransitGatewayRouting string

const (
	TransitGatewayRoutingLocal  TransitGatewayRouting = "Local"
	TransitGatewayRoutingGlobal TransitGatewayRouting = "Global"
)

type VPCLoadBalancerVisibility string

const (
	VPCLoadBalancerVisibilityPublic  VPCLoadBalancerVisibility = "Public"
	VPCLoadBalancerVisibilityPrivate VPCLoadBalancerVisibility = "Private"
)

func init() {
	objectTypes = append(objectTypes, &IBMPowerVSCluster{}, &IBMPowerVSClusterList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=ibmpowervsclusters,scope=Namespaced,categories=cluster-api
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this IBMPowerVSCluster belongs"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of IBMPowerVSCluster"
// +kubebuilder:printcolumn:name="Endpoint",type="string",priority=1,JSONPath=".spec.controlPlaneEndpoint.host",description="Control Plane Endpoint"
// +kubebuilder:printcolumn:name="Port",type="string",priority=1,JSONPath=".spec.controlPlaneEndpoint.port",description="Control Plane Port"

// IBMPowerVSClusterSpec defines the desired state of IBMPowerVSCluster.
type IBMPowerVSCluster struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of IBMPowerVSCluster
	// +required
	Spec IBMPowerVSClusterSpec `json:"spec"`

	// status defines the observed state of IBMPowerVSCluster
	// +optional
	Status IBMPowerVSClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// IBMPowerVSClusterList contains a list of IBMPowerVSCluster.
type IBMPowerVSClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []IBMPowerVSCluster `json:"items"`
}

// IBMPowerVSClusterSpec defines the desired state of IBMPowerVSCluster.
//
// Topology Rules:
// 1. If VIP, forbid all VPC-related fields.
// +kubebuilder:validation:XValidation:rule="self.topology == 'LoadBalancer' || (!has(self.vpc) && !has(self.vpcSubnets) && !has(self.vpcSecurityGroups) && !has(self.loadBalancers) && !has(self.transitGateway))",message="VPC, VPC subnets, VPC security groups, load balancers, and transit gateway can only be configured when topology is LoadBalancer"
//
// 2. If VIP, mandate Workspace and Network, AND force them to be 'Reference' only (no provisioning).
// +kubebuilder:validation:XValidation:rule="self.topology == 'LoadBalancer' || (has(self.workspace) && self.workspace.type == 'Reference' && has(self.network) && self.network.type == 'Reference')",message="workspace and network are required when topology is VIP, and workspace/network must be set to 'Reference' (provisioning is not supported in VIP mode)"
//
// 3. If LoadBalancer, mandate the VPC field.
// +kubebuilder:validation:XValidation:rule="self.topology == 'VIP' || has(self.vpc)",message="VPC configuration is required when topology is LoadBalancer"
type IBMPowerVSClusterSpec struct {
	// controlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint APIEndpoint `json:"controlPlaneEndpoint,omitempty,omitzero"`

	// topology defines the architectural mode for external cluster access.
	// +kubebuilder:validation:Enum=VIP;LoadBalancer
	Topology ClusterTopology `json:"topology,omitempty"`

	// zone is the name of the PowerVS zone where the cluster will be created.
	// Possible values can be found at https://cloud.ibm.com/docs/power-iaas?topic=power-iaas-creating-power-virtual-server.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="zone is immutable"
	// +kubebuilder:validation:MinLength=1
	Zone string `json:"zone,omitempty"`

	// +optional
	// resourceGroup identifies the existing IBM Cloud Resource Group under which the resources will be created.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="resourceGroup is immutable"
	ResourceGroup ResourceGroup `json:"resourceGroup,omitempty,omitzero"`

	// workspace specifies how the PowerVS workspace is sourced.
	// A PowerVS workspace is a container for PowerVS resources in a specific zone.
	// More details: https://cloud.ibm.com/docs/power-iaas?topic=power-iaas-creating-power-virtual-server
	// +optional
	Workspace WorkspaceSource `json:"workspace,omitempty,omitzero"`

	// network specifies how the PowerVS network should be sourced.
	// +optional
	Network NetworkSource `json:"network,omitempty,omitzero"`

	// transitGateway specifies how the IBM Cloud Transit Gateway should be sourced.
	// IBM Cloud Transit Gateway helps in establishing network connectivity between IBM Cloud Power VS and VPC infrastructure
	// More information about TransitGateway can be found here https://www.ibm.com/products/transit-gateway.
	// +optional
	TransitGateway TransitGateway `json:"transitGateway,omitempty,omitzero"`

	// vpc contains information about IBM Cloud VPC resources.
	// +optional
	VPC VPCSource `json:"vpc,omitempty,omitzero"`

	// vpcSubnets specifies the subnets to use within the IBM Cloud VPC.
	// when omitted system will create the subnets in all the zone corresponding to VPC.Region, with name CLUSTER_NAME-vpcsubnet-ZONE_NAME.
	// possible values can be found here https://cloud.ibm.com/docs/power-iaas?topic=power-iaas-creating-power-virtual-server.
	// +optional
	// +listType=atomic
	VPCSubnets []VPCSubnet `json:"vpcSubnets,omitempty,omitzero"`

	// vpcSecurityGroups specifies the security groups to be associated with the VPC.
	// +optional
	// +listType=atomic
	VPCSecurityGroups []VPCSecurityGroup `json:"vpcSecurityGroups,omitempty,omitzero"`

	// loadBalancers is the configuration for VPC Load Balancers.
	// If omitted, a default public load balancer will be created for the control plane.
	// +optional
	// +listType=atomic
	LoadBalancers []VPCLoadBalancer `json:"loadBalancers,omitempty,omitzero"`

	// cosInstance contains options to configure a supporting IBM Cloud COS bucket for this
	// cluster - currently used for nodes requiring Ignition
	// (https://coreos.github.io/ignition/) for bootstrapping (requires
	// BootstrapFormatIgnition feature flag to be enabled).
	// +optional
	CosInstance CosInstance `json:"cosInstance,omitempty,omitzero"`

	// ignition defined options related to the bootstrapping systems where Ignition is used.
	// +optional
	Ignition Ignition `json:"ignition,omitempty,omitzero"`
}

// IBMPowerVSClusterStatus defines the observed state of IBMPowerVSCluster.
type IBMPowerVSClusterStatus struct {
	// conditions represents the observations of a IBMPowerVSCluster's current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=32
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// initialization provides observations of the IBMPowerVSCluster initialization process.
	// NOTE: Fields in this struct are part of the Cluster API contract and are used to orchestrate initial Cluster provisioning.
	// +optional
	Initialization IBMPowerVSClusterInitializationStatus `json:"initialization,omitempty,omitzero"`

	// resourceGroup is the reference to the Power VS resource group.
	// +optional
	ResourceGroup ResourceReference `json:"resourceGroup,omitempty,omitzero"`

	// workspace is the reference to the PowerVS workspace.
	// +optional
	Workspace ResourceReference `json:"workspace,omitempty,omitzero"`

	// network is the reference to the Power VS network used for this cluster.
	// +optional
	Network ResourceReference `json:"network,omitempty,omitzero"`

	// dhcpServer is the reference to the Power VS DHCP server.
	// +optional
	DHCPServer ResourceReference `json:"dhcpServer,omitempty,omitzero"`

	// vpc is the reference to the IBM Cloud VPC resources.
	// +optional
	VPC ResourceReference `json:"vpc,omitempty,omitzero"`

	// vpcSubnets is a list of references to IBM Cloud VPC subnets.
	// +optional
	// +listType=map
	// +listMapKey=name
	VPCSubnets []ResourceReference `json:"vpcSubnets,omitempty,omitzero"`

	// vpcSecurityGroups is a list of observed IBM Cloud VPC security groups.
	// +optional
	// +listType=map
	// +listMapKey=name
	VPCSecurityGroups []VPCSecurityGroupStatus `json:"vpcSecurityGroups,omitempty,omitzero"`

	// loadBalancers is a list of observed IBM Cloud VPC load balancers.
	// +optional
	// +listType=map
	// +listMapKey=name
	LoadBalancers []VPCLoadBalancerStatus `json:"loadBalancers,omitempty,omitzero"`

	// transitGateway is the reference to the IBM Cloud TransitGateway.
	// +optional
	TransitGateway TransitGatewayStatus `json:"transitGateway,omitempty,omitzero"`

	// cosInstance is the reference to the IBM Cloud COS Instance resource.
	// +optional
	COSInstance ResourceReference `json:"cosInstance,omitempty,omitzero"`

	// deprecated groups all the status fields that are deprecated.
	// +optional
	Deprecated *IBMPowerVSClusterDeprecatedStatus `json:"deprecated,omitempty"`
}

// APIEndpoint represents a reachable Kubernetes API endpoint.
// +kubebuilder:validation:MinProperties=1
type APIEndpoint struct {
	// host is the hostname on which the API server is serving.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	Host string `json:"host,omitempty"`

	// port is the port on which the API server is serving.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`
}

// IBMPowerVSClusterInitializationStatus provides observations of the IBMPowerVSCluster initialization process.
// +kubebuilder:validation:MinProperties=1
type IBMPowerVSClusterInitializationStatus struct {
	// provisioned is true when the infrastructure provider reports that the Cluster's infrastructure is fully provisioned.
	// NOTE: this field is part of the Cluster API contract, and it is used to orchestrate initial Cluster provisioning.
	// +optional
	Provisioned *bool `json:"provisioned,omitempty"`
}

// ResourceReference identifies a resource with id.
type ResourceReference struct {
	// id represents the id of the resource.
	// +optional
	ID string `json:"id,omitempty"`

	// name is the name of the resource.
	// When used in a list, this field acts as the unique correlation key (listMapKey)
	// to map the Status object back to its corresponding Spec definition.
	// +optional
	Name string `json:"name,omitempty"`
}

// TransitGatewayStatus defines the observed state of the transit gateway and its connections.
type TransitGatewayStatus struct {
	// id represents the id of the Transit Gateway.
	// +optional
	ID string `json:"id,omitempty"`

	// vpcConnection defines the observed VPC connection in the transit gateway.
	// +optional
	VPCConnection ResourceReference `json:"vpcConnection,omitempty,omitzero"`

	// powerVSConnection defines the observed PowerVS connection in the transit gateway.
	// +optional
	PowerVSConnection ResourceReference `json:"powerVSConnection,omitempty,omitzero"`
}

// VPCSecurityGroupStatus defines the observed state of a VPC Security Group and its rules.
type VPCSecurityGroupStatus struct {
	// name is the name of the security group.
	// When used in a list, this field acts as the unique correlation key (listMapKey)
	// to map the Status object back to its corresponding Spec definition.
	// +optional
	Name string `json:"name,omitempty"`

	// id represents the id of the security group.
	// +optional
	ID string `json:"id,omitempty"`

	// ruleIDs contains the IDs of the rules created under the security group.
	// +optional
	RuleIDs []string `json:"ruleIDs,omitempty"`
}

// VPCLoadBalancerStatus defines the observed state of a VPC load balancer.
type VPCLoadBalancerStatus struct {
	// name is the name of the load balancer.
	// When used in a list, this field acts as the unique correlation key (listMapKey)
	// to map the Status object back to its corresponding Spec definition.
	// +optional
	Name string `json:"name,omitempty"`

	// id is the ID of the VPC load balancer.
	// +optional
	ID string `json:"id,omitempty"`

	// state is the status of the load balancer.
	// +optional
	State VPCLoadBalancerState `json:"state,omitempty"`

	// hostname is the hostname of the load balancer.
	// +optional
	Hostname string `json:"hostname,omitempty"`
}

// IBMPowerVSClusterDeprecatedStatus groups all the status fields that are deprecated and will be removed in a future version.
// See https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20240916-improve-status-in-CAPI-resources.md for more context.
type IBMPowerVSClusterDeprecatedStatus struct {
	// v1beta2 groups all the status fields that are deprecated and will be removed when support for v1beta2 will be dropped.
	// +optional
	V1Beta2 *IBMPowerVSClusterV1Beta2DeprecatedStatus `json:"v1beta2,omitempty"`
}

// IBMPowerVSClusterV1Beta2DeprecatedStatus groups all the status fields that are deprecated and will be removed when support for v1beta1 will be dropped.
// See https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20240916-improve-status-in-CAPI-resources.md for more context.
type IBMPowerVSClusterV1Beta2DeprecatedStatus struct {
	// conditions defines current service state of the VSphereCluster.
	//
	// Deprecated: This field is deprecated and is going to be removed when support for v1beta1 will be dropped. Please see https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20240916-improve-status-in-CAPI-resources.md for more details.
	//
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// GetConditions returns the observations of the operational state of the IBMPowerVSCluster resource.
func (r *IBMPowerVSCluster) GetConditions() []metav1.Condition {
	return r.Status.Conditions
}

// SetConditions sets conditions for an API object.
func (r *IBMPowerVSCluster) SetConditions(conditions []metav1.Condition) {
	r.Status.Conditions = conditions
}

// GetV1Beta1Conditions returns the set of conditions for this object.
func (r *IBMPowerVSCluster) GetV1Beta1Conditions() clusterv1.Conditions {
	if r.Status.Deprecated == nil || r.Status.Deprecated.V1Beta2 == nil {
		return nil
	}
	return r.Status.Deprecated.V1Beta2.Conditions
}

// SetV1Beta1Conditions sets conditions for an API object.
func (r *IBMPowerVSCluster) SetV1Beta1Conditions(conditions clusterv1.Conditions) {
	if r.Status.Deprecated == nil {
		r.Status.Deprecated = &IBMPowerVSClusterDeprecatedStatus{}
	}
	if r.Status.Deprecated.V1Beta2 == nil {
		r.Status.Deprecated.V1Beta2 = &IBMPowerVSClusterV1Beta2DeprecatedStatus{}
	}
	r.Status.Deprecated.V1Beta2.Conditions = conditions
}

// WorkspaceSource defines how the PowerVS workspace is sourced.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? has(self.provision) : !has(self.provision)",message="provision configuration is required when type is Provision, and forbidden otherwise"
type WorkspaceSource struct {
	// type defines how the workspace is sourced.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="workspace type is immutable once set"
	Type SourceType `json:"type"`

	// reference tells the controller to use an existing PowerVS workspace.
	// Supported identifiers are name and id.
	// If more than one workspace has the same name, use id.
	// +optional
	Reference ResourceIdentifier `json:"reference,omitempty,omitzero"`

	// provision defines the configuration for creating a new PowerVS workspace.
	// +optional
	Provision WorkspaceProvisionConfig `json:"provision,omitempty,omitzero"`
}

// WorkspaceProvisionConfig defines the parameters for creating a new workspace.
type WorkspaceProvisionConfig struct {
	// name is the explicit name of the workspace to be created.
	// If omitted, the system will dynamically create the workspace with the name <CLUSTER_NAME>-workspace.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
}

// ResourceIdentifier defines the identification of a specific PowerVS resource by ID or Name.
// +kubebuilder:validation:XValidation:rule="(has(self.id) ? 1 : 0) + (has(self.name) ? 1 : 0) == 1",message="exactly one of id or name must be specified"
type ResourceIdentifier struct {
	// ID of the resource.
	// +optional
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id,omitempty"`

	// Name of the resource.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
}

// NetworkSource defines how to source the PowerVS network.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? has(self.provision) : !has(self.provision)",message="provision configuration is required when type is Provision, and forbidden otherwise"
type NetworkSource struct {
	// type defines how the Network is sourced.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Network type is immutable once set"
	Type SourceType `json:"type,omitempty"`

	// reference tells the controller to look up an EXISTING PowerVS network.
	// +optional
	Reference ResourceIdentifier `json:"reference,omitempty,omitzero"`

	// provision provides the configuration for the controller to CREATE a new Network and DHCP Server.
	// +optional
	Provision NetworkProvisionConfig `json:"provision,omitempty,omitzero"`
}

// NetworkProvisionConfig defines the parameters for creating a new PowerVS Network.
type NetworkProvisionConfig struct {
	// dhcpServer contains the configuration for the DHCP server that will be created.
	// +optional
	DHCPServer DHCPServer `json:"dhcpServer,omitempty,omitzero"`
}

// DHCPServer contains the configuration for a NEW DHCP server.
type DHCPServer struct {
	// name is the name of the DHCP Service to be created. Only alphanumeric characters and dashes are allowed.
	// If omitted, the name will default to DHCPSERVER<CLUSTER_NAME>_Private.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// CIDR is the CIDR for the DHCP private network.
	// +optional
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}($|/[0-9]{1,2})$`
	CIDR string `json:"cidr,omitempty"`

	// DNSServer is the DNS Server for the DHCP service.
	// +optional
	DNSServer string `json:"dnsServer,omitempty"`

	// snat indicates the SNAT policy for the DHCP service.
	// Allowed values are "Enabled" and "Disabled".
	// If omitted, the system will choose a Enabled policy by default.
	// +optional
	// +kubebuilder:validation:Enum=Enabled;Disabled
	Snat DHCPSnatPolicy `json:"snat,omitempty"`
}

// ResourceGroup defines the identification of an existing IBM Cloud Resource Group.
// +kubebuilder:validation:XValidation:rule="(has(self.id) ? 1 : 0) + (has(self.name) ? 1 : 0) == 1",message="exactly one of id or name must be specified"
type ResourceGroup struct {
	// ID is the ID of the existing Resource Group.
	// +optional
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id,omitempty"`

	// Name is the name of the existing Resource Group.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
}

// VPCSource defines how to source the IBM Cloud VPC.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? has(self.provision) : !has(self.provision)",message="provision configuration is required when type is Provision, and forbidden otherwise"
type VPCSource struct {
	// Type defines how the VPC is sourced.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VPC type is immutable once set"
	Type SourceType `json:"type,omitempty"`

	// Region is the IBM Cloud region for the VPC.
	// This is required for both Reference and Provision types to initialize the VPC client.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VPC region is immutable once set"
	Region string `json:"region,omitempty"`

	// Reference identifies an EXISTING VPC.
	// +optional
	Reference ResourceIdentifier `json:"reference,omitempty,omitzero"`

	// Provision contains settings for CREATING a new VPC.
	// +optional
	Provision VPCProvisionConfig `json:"provision,omitempty,omitzero"`
}

// VPCProvisionConfig defines the parameters for creating a new IBM Cloud VPC.
type VPCProvisionConfig struct {
	// Name is the explicit name of the VPC to be created.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^([a-z]|[a-z][-a-z0-9]*[a-z0-9])$`
	Name string `json:"name,omitempty"`
}

// VPCSubnet defines the sourcing strategy for a VPC subnet.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? has(self.provision) : !has(self.provision)",message="provision configuration is required when type is Provision, and forbidden otherwise"
type VPCSubnet struct {
	// Type defines how the subnet is sourced.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	Type SourceType `json:"type,omitempty"`

	// Reference identifies an EXISTING VPC Subnet.
	// +optional
	Reference ResourceIdentifier `json:"reference,omitempty,omitzero"`

	// Provision contains settings for CREATING a new VPC Subnet.
	// +optional
	Provision VPCSubnetProvisionConfig `json:"provision,omitempty,omitzero"`
}

// VPCSubnetProvisionConfig defines the parameters for creating a new VPC Subnet.
type VPCSubnetProvisionConfig struct {
	// Name is the name of the subnet to be created.
	// If omitted, the name will default to <CLUSTER_NAME>-vpcsubnet-<INDEX>.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^([a-z]|[a-z][-a-z0-9]*[a-z0-9])$`
	Name string `json:"name,omitempty"`

	// CIDR is the IPv4 CIDR block for the subnet.
	// +optional
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}($|/[0-9]{1,2})$`
	CIDR string `json:"cidr,omitempty"`

	// Zone is the zone where the subnet should be created.
	// If omitted, a random zone is picked from the VPC region.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Zone string `json:"zone,omitempty"`
}

// VPCSecurityGroup defines a VPC Security Group strategy.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? has(self.provision) : !has(self.provision)",message="provision configuration is required when type is Provision, and forbidden otherwise"
type VPCSecurityGroup struct {
	// Type defines how the Security Group is sourced.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	Type SourceType `json:"type,omitempty"`

	// Reference identifies an EXISTING Security Group.
	// +optional
	Reference ResourceIdentifier `json:"reference,omitempty,omitzero"`

	// Provision contains settings for CREATING a new Security Group.
	// +optional
	Provision VPCSecurityGroupProvisionConfig `json:"provision,omitempty,omitzero"`
}

// VPCSecurityGroupProvisionConfig defines the parameters for creating a new Security Group.
type VPCSecurityGroupProvisionConfig struct {
	// Name is the name of the Security Group to be created.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// Rules are the Security Group Rules to be created for this Security Group.
	// +optional
	Rules []VPCSecurityGroupRule `json:"rules,omitempty,omitzero"`

	// Tags are the IBM Cloud tags to be added to the Security Group.
	// +optional
	Tags []string `json:"tags,omitempty"`
}

// VPCSecurityGroupRule defines a VPC Security Group Rule.
// +kubebuilder:validation:XValidation:rule="(has(self.destination) && !has(self.source)) || (!has(self.destination) && has(self.source))",message="exactly one of destination or source must be provided"
// +kubebuilder:validation:XValidation:rule="self.direction == 'inbound' ? has(self.source) && !has(self.destination) : true",message="inbound rules must have source and no destination"
// +kubebuilder:validation:XValidation:rule="self.direction == 'outbound' ? has(self.destination) && !has(self.source) : true",message="outbound rules must have destination and no source"
type VPCSecurityGroupRule struct {
	// Action defines whether to allow or deny traffic.
	// +required
	Action VPCSecurityGroupRuleAction `json:"action"`

	// Direction defines whether the traffic is inbound or outbound.
	// +required
	Direction VPCSecurityGroupRuleDirection `json:"direction"`

	// Source defines the source of inbound traffic.
	// +optional
	Source *VPCSecurityGroupRulePrototype `json:"source,omitempty"`

	// Destination defines the destination of outbound traffic.
	// +optional
	Destination *VPCSecurityGroupRulePrototype `json:"destination,omitempty"`
}

// VPCSecurityGroupRulePrototype defines a VPC Security Group Rule's traffic specifics.
// +kubebuilder:validation:XValidation:rule="self.protocol != 'icmp' ? (!has(self.icmpCode) && !has(self.icmpType)) : true",message="icmpCode and icmpType are only supported for icmp protocol"
// +kubebuilder:validation:XValidation:rule="(self.protocol == 'all' || self.protocol == 'icmp') ? !has(self.portRange) : true",message="portRange is not valid for 'all' or 'icmp' protocols"
type VPCSecurityGroupRulePrototype struct {
	// Protocol defines the traffic protocol used for the Security Group Rule.
	// +required
	Protocol VPCSecurityGroupRuleProtocol `json:"protocol"`

	// Remotes is a set of VPCSecurityGroupRuleRemote's that define the traffic allowed by the Rule.
	// +required
	// +kubebuilder:validation:MinItems=1
	Remotes []VPCSecurityGroupRuleRemote `json:"remotes"`

	// PortRange is a range of ports allowed for the Rule's remote.
	// +optional
	PortRange *VPCSecurityGroupPortRange `json:"portRange,omitempty"`

	// ICMPCode is the ICMP code for the Rule.
	// Only used when Protocol is icmp.
	// +optional
	ICMPCode int64 `json:"icmpCode,omitempty"`

	// ICMPType is the ICMP type for the Rule.
	// Only used when Protocol is icmp.
	// +optional
	ICMPType int64 `json:"icmpType,omitempty"`
}

// VPCSecurityGroupRuleRemote defines the source or destination for a security group rule.
// +kubebuilder:validation:XValidation:rule="self.remoteType == 'any' ? (!has(self.cidrSubnetName) && !has(self.address) && !has(self.securityGroupName)) : true",message="cidrSubnetName, address, and securityGroupName must not be set when remoteType is any"
// +kubebuilder:validation:XValidation:rule="self.remoteType == 'cidr' ? has(self.cidrSubnetName) : !has(self.cidrSubnetName)",message="cidrSubnetName is required when remoteType is cidr, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.remoteType == 'address' ? has(self.address) : !has(self.address)",message="address is required when remoteType is address, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.remoteType == 'sg' ? has(self.securityGroupName) : !has(self.securityGroupName)",message="securityGroupName is required when remoteType is sg, and forbidden otherwise"
type VPCSecurityGroupRuleRemote struct {
	// RemoteType defines the type of filter to define for the remote's destination/source.
	// +required
	// +kubebuilder:validation:Enum=any;cidr;address;sg
	RemoteType VPCSecurityGroupRuleRemoteType `json:"remoteType"`

	// CIDRSubnetName is the name of the VPC Subnet to retrieve the CIDR from.
	// +optional
	CIDRSubnetName string `json:"cidrSubnetName,omitempty"`

	// Address is the IP address to use for the remote.
	// +optional
	Address string `json:"address,omitempty"`

	// SecurityGroupName is the name of the VPC Security Group to use for the remote.
	// +optional
	SecurityGroupName string `json:"securityGroupName,omitempty"`
}

// VPCSecurityGroupPortRange represents a range of ports, minimum to maximum.
// +kubebuilder:validation:XValidation:rule="self.maximumPort >= self.minimumPort",message="maximum port must be greater than or equal to minimum port"
type VPCSecurityGroupPortRange struct {
	// maximumPort is the inclusive upper range of ports.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	MaximumPort int64 `json:"maximumPort"`

	// minimumPort is the inclusive lower range of ports.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	MinimumPort int64 `json:"minimumPort"`
}

// VPCLoadBalancer defines the sourcing strategy for a VPC Load Balancer.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? has(self.provision) : !has(self.provision)",message="provision configuration is required when type is Provision, and forbidden otherwise"
type VPCLoadBalancer struct {
	// Type defines how the Load Balancer is sourced.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	Type SourceType `json:"type,omitempty"`

	// Reference identifies an EXISTING VPC Load Balancer.
	// +optional
	Reference ResourceIdentifier `json:"reference,omitempty,omitzero"`

	// Provision contains settings for CREATING a new VPC Load Balancer.
	// +optional
	Provision VPCLoadBalancerProvisionConfig `json:"provision,omitempty,omitzero"`
}

// VPCLoadBalancerProvisionConfig defines the parameters for creating a new VPC Load Balancer.
type VPCLoadBalancerProvisionConfig struct {
	// name is the name of the Load Balancer to be created.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^([a-z]|[a-z][-a-z0-9]*[a-z0-9])$`
	Name string `json:"name,omitempty"`

	// visibility indicates whether the load balancer is public or private.
	// Allowed values are "Public" and "Private".
	// If omitted, a default visibility will be selected by the controller.
	// +optional
	// +kubebuilder:validation:Enum=Public;Private
	Visibility VPCLoadBalancerVisibility `json:"visibility,omitempty"`

	// additionalListeners sets the additional listeners for the load balancer.
	// +optional
	// +listType=map
	// +listMapKey=port
	AdditionalListeners []AdditionalListenerSpec `json:"additionalListeners,omitempty,omitzero"`

	// backendPools defines the load balancer's backend pools.
	// +optional
	BackendPools []VPCLoadBalancerBackendPoolSpec `json:"backendPools,omitempty,omitzero"`

	// securityGroups defines the IDs or Names of existing Security Groups to attach.
	// +optional
	SecurityGroups []ResourceIdentifier `json:"securityGroups,omitempty,omitzero"`

	// subnets defines the IDs or Names of existing VPC Subnets to attach.
	// +optional
	Subnets []ResourceIdentifier `json:"subnets,omitempty,omitzero"`
}

// VPCLoadBalancerBackendPoolSpec defines the desired configuration of a VPC Load Balancer Backend Pool.
type VPCLoadBalancerBackendPoolSpec struct {
	// Name defines the name of the Backend Pool.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^([a-z]|[a-z][-a-z0-9]*[a-z0-9])$`
	Name string `json:"name,omitempty"`

	// Algorithm defines the load balancing algorithm to use.
	// +required
	Algorithm VPCLoadBalancerBackendPoolAlgorithm `json:"algorithm"`

	// HealthMonitor defines the backend pool's health monitor.
	// +required
	HealthMonitor VPCLoadBalancerHealthMonitorSpec `json:"healthMonitor"`

	// Protocol defines the protocol to use for the Backend Pool.
	// +required
	Protocol VPCLoadBalancerBackendPoolProtocol `json:"protocol"`
}

// AdditionalListenerSpec defines the desired state of an additional listener on a VPC load balancer.
type AdditionalListenerSpec struct {
	// DefaultPoolName defines the name of a VPC Load Balancer Backend Pool to use for the Listener.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^([a-z]|[a-z][-a-z0-9]*[a-z0-9])$`
	DefaultPoolName string `json:"defaultPoolName,omitempty"`

	// Port sets the port for the additional listener.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int64 `json:"port"`

	// Protocol defines the protocol to use for the VPC Load Balancer Listener.
	// Will default to TCP protocol if not specified.
	// +optional
	Protocol VPCLoadBalancerListenerProtocol `json:"protocol,omitempty"`

	// Selector is used to find IBMPowerVSMachines with matching labels.
	// If the label matches, the machine is then added to the load balancer listener configuration.
	// +optional
	Selector metav1.LabelSelector `json:"selector,omitempty"`
}

// VPCLoadBalancerHealthMonitorSpec defines the health check configuration.
// +kubebuilder:validation:XValidation:rule="self.delay > self.timeout",message="delay must be greater than timeout"
type VPCLoadBalancerHealthMonitorSpec struct {
	// Delay defines the seconds to wait between health checks.
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=60
	Delay int64 `json:"delay"`

	// Timeout defines the seconds to wait for a response.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=59
	Timeout int64 `json:"timeout"`

	// Retries defines the max retries for health check.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	Retries int64 `json:"retries"`

	// Type defines the protocol used for health checks.
	// +required
	Type VPCLoadBalancerBackendPoolHealthMonitorType `json:"type"`

	// Port defines the port to perform health monitoring on.
	// +optional
	Port int64 `json:"port,omitempty"`

	// URLPath defines the URL to use for health monitoring.
	// +optional
	URLPath string `json:"urlPath,omitempty"`
}

// TransitGateway defines the sourcing strategy for an IBM Cloud Transit Gateway.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? has(self.provision) : !has(self.provision)",message="provision configuration is required when type is Provision, and forbidden otherwise"
type TransitGateway struct {
	// type defines how the Transit Gateway is sourced.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	Type SourceType `json:"type,omitempty"`

	// reference identifies an EXISTING Transit Gateway and how its connections should be managed.
	// +optional
	Reference TransitGatewayReferenceConfig `json:"reference,omitempty,omitzero"`

	// provision contains settings for CREATING a new Transit Gateway.
	// +optional
	Provision TransitGatewayProvisionConfig `json:"provision,omitempty,omitzero"`
}

// TransitGatewayProvisionConfig defines the parameters for creating a new Transit Gateway.
type TransitGatewayProvisionConfig struct {
	// Name is the explicit name of the Transit Gateway to be created.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^([a-zA-Z]|[a-zA-Z][-_a-zA-Z0-9]*[a-zA-Z0-9])$`
	Name string `json:"name,omitempty"`

	// routing indicates whether to use global or local routing for the transit gateway.
	// Allowed values are "Local" and "Global".
	// Set this field to "Global" only when PowerVS and VPC are from different regions.
	// If omitted, the system will dynamically decide based on the PowerVS Zone and VPC Region.
	// +optional
	// +kubebuilder:validation:Enum=Local;Global
	Routing TransitGatewayRouting `json:"routing,omitempty"`
}

// TransitGatewayReferenceConfig defines the parameters for referencing an existing Transit Gateway.
// +kubebuilder:validation:XValidation:rule="(has(self.id) ? 1 : 0) + (has(self.name) ? 1 : 0) == 1",message="exactly one of id or name must be specified"
type TransitGatewayReferenceConfig struct {
	// id of the existing Transit Gateway.
	// +optional
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id,omitempty"`

	// name of the existing Transit Gateway.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// vpcConnection defines how the controller manages the VPC connection to this Transit Gateway.
	// +optional
	VPCConnection TransitGatewayConnection `json:"vpcConnection,omitempty,omitzero"`

	// powerVSConnection defines how the controller manages the PowerVS workspace connection to this Transit Gateway.
	// +optional
	PowerVSConnection TransitGatewayConnection `json:"powerVSConnection,omitempty,omitzero"`
}

// TransitGatewayConnection defines the sourcing strategy for a Transit Gateway connection.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? true : !has(self.provision)",message="provision configuration is forbidden when type is Reference"
type TransitGatewayConnection struct {
	// type defines how the connection is managed.
	// Reference means the connection already exists and you must provide the ID/Name.
	// Provision means the controller will create a new connection.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	// +kubebuilder:default=Provision
	Type SourceType `json:"type"`

	// reference identifies an EXISTING transit gateway connection.
	// +optional
	Reference ResourceIdentifier `json:"reference,omitempty,omitzero"`

	// provision contains settings for CREATING a new transit gateway connection.
	// +optional
	Provision TransitGatewayConnectionProvisionConfig `json:"provision,omitempty,omitzero"`
}

// TransitGatewayConnectionProvisionConfig defines the parameters for creating a new TG connection.
type TransitGatewayConnectionProvisionConfig struct {
	// name is the explicit name of the connection to be created.
	// If omitted, a default name will be generated by the controller.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
}

// CosInstance defines the sourcing strategy for an IBM Cloud COS instance and bucket.
// This is primarily used for nodes requiring Ignition for bootstrapping.
// +kubebuilder:validation:XValidation:rule="self.type == 'Reference' ? has(self.reference) : !has(self.reference)",message="reference configuration is required when type is Reference, and forbidden otherwise"
// +kubebuilder:validation:XValidation:rule="self.type == 'Provision' ? has(self.provision) : !has(self.provision)",message="provision configuration is required when type is Provision, and forbidden otherwise"
type CosInstance struct {
	// Type defines how the COS instance is sourced.
	// +required
	// +kubebuilder:validation:Enum=Reference;Provision
	Type SourceType `json:"type,omitempty"`

	// BucketRegion is the IBM Cloud region where the COS bucket resides or will be created.
	// This is required to initialize the COS client.
	// +optional
	// +kubebuilder:validation:MinLength=1
	BucketRegion string `json:"bucketRegion,omitempty"`

	// Reference identifies an EXISTING COS instance and bucket.
	// +optional
	Reference COSReference `json:"reference,omitempty,omitzero"`

	// Provision contains settings for CREATING a new COS instance and bucket.
	// +optional
	Provision COSProvisionConfig `json:"provision,omitempty,omitzero"`
}

// COSReference identifies an existing COS instance and bucket.
type COSReference struct {
	// Instance identifies the existing COS instance.
	// +required
	Instance ResourceIdentifier `json:"instance"`

	// BucketName is the name of the existing IBM Cloud COS bucket.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	BucketName string `json:"bucketName"`
}

// COSProvisionConfig defines parameters for creating a new COS instance and bucket.
type COSProvisionConfig struct {
	// Name defines the name of the IBM Cloud COS instance to be created.
	// +optional
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`
	Name string `json:"name,omitempty"`

	// BucketName is the name of the IBM Cloud COS bucket to be created.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	BucketName string `json:"bucketName,omitempty"`
}

// Ignition defines options related to the bootstrapping systems where Ignition is used.
type Ignition struct {
	// version defines which version of Ignition will be used to generate bootstrap data.
	//
	// +optional
	// +kubebuilder:validation:Enum="2.3";"2.4";"3.0";"3.1";"3.2";"3.3";"3.4"
	Version string `json:"version,omitempty"`
}
