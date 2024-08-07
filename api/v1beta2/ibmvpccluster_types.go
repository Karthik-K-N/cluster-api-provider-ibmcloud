/*
Copyright 2022 The Kubernetes Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// ClusterFinalizer allows DockerClusterReconciler to clean up resources associated with DockerCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "ibmvpccluster.infrastructure.cluster.x-k8s.io"
)

// IBMVPCClusterSpec defines the desired state of IBMVPCCluster.
type IBMVPCClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The IBM Cloud Region the cluster lives in.
	Region string `json:"region"`

	// The VPC resources should be created under the resource group.
	ResourceGroup string `json:"resourceGroup"`

	// The Name of VPC.
	VPC string `json:"vpc,omitempty"`

	// The Name of availability zone.
	Zone string `json:"zone,omitempty"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint capiv1beta1.APIEndpoint `json:"controlPlaneEndpoint"`

	// ControlPlaneLoadBalancer is optional configuration for customizing control plane behavior.
	// +optional
	ControlPlaneLoadBalancer *VPCLoadBalancerSpec `json:"controlPlaneLoadBalancer,omitempty"`

	// network represents the VPC network to use for the cluster.
	// +optional
	Network *VPCNetworkSpec `json:"network,omitempty"`
}

// VPCLoadBalancerSpec defines the desired state of an VPC load balancer.
type VPCLoadBalancerSpec struct {
	// Name sets the name of the VPC load balancer.
	// +kubebuilder:validation:MinLength:=1
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Pattern=`^([a-z]|[a-z][-a-z0-9]*[a-z0-9])$`
	// +optional
	Name string `json:"name,omitempty"`

	// id of the loadbalancer
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength:=64
	// +kubebuilder:validation:Pattern=`^[-0-9a-z_]+$`
	// +optional
	ID *string `json:"id,omitempty"`

	// public indicates that load balancer is public or private
	// +kubebuilder:default=true
	// +optional
	Public *bool `json:"public,omitempty"`

	// AdditionalListeners sets the additional listeners for the control plane load balancer.
	// +listType=map
	// +listMapKey=port
	// +optional
	// ++kubebuilder:validation:UniqueItems=true
	AdditionalListeners []AdditionalListenerSpec `json:"additionalListeners,omitempty"`
}

// AdditionalListenerSpec defines the desired state of an
// additional listener on an VPC load balancer.
type AdditionalListenerSpec struct {
	// Port sets the port for the additional listener.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int64 `json:"port"`
}

// VPCNetworkSpec defines the desired state of the network resources for the cluster for extended VPC Infrastructure support.
type VPCNetworkSpec struct {
	// workerSubnets is a set of Subnet's which define the Worker subnets.
	// +optional
	WorkerSubnets []Subnet `json:"workerSubnets,omitempty"`

	// controlPlaneSubnets is a set of Subnet's which define the Control Plane subnets.
	// +optional
	ControlPlaneSubnets []Subnet `json:"controlPlaneSubnets,omitempty"`

	// resourceGroup is the name of the Resource Group containing all of the newtork resources.
	// This can be different than the Resource Group containing the remaining cluster resources.
	// +optional
	ResourceGroup *string `json:"resourceGroup,omitempty"`

	// vpc defines the IBM Cloud VPC for extended VPC Infrastructure support.
	// +optional
	VPC *VPCResource `json:"vpc,omitempty"`

	// TODO(cjschaef): Complete spec definition (SecurityGroups, etc.)
}

// VPCSecurityGroupStatus defines a vpc security group resource status with its id and respective rule's ids.
type VPCSecurityGroupStatus struct {
	// id represents the id of the resource.
	ID *string `json:"id,omitempty"`
	// rules contains the id of rules created under the security group
	RuleIDs []*string `json:"ruleIDs,omitempty"`
	// +kubebuilder:default=false
	// controllerCreated indicates whether the resource is created by the controller.
	ControllerCreated *bool `json:"controllerCreated,omitempty"`
}

// VPCLoadBalancerStatus defines the status VPC load balancer.
type VPCLoadBalancerStatus struct {
	// id of VPC load balancer.
	// +optional
	ID *string `json:"id,omitempty"`
	// State is the status of the load balancer.
	State VPCLoadBalancerState `json:"state,omitempty"`
	// hostname is the hostname of load balancer.
	// +optional
	Hostname *string `json:"hostname,omitempty"`
	// +kubebuilder:default=false
	// controllerCreated indicates whether the resource is created by the controller.
	ControllerCreated *bool `json:"controllerCreated,omitempty"`
}

// IBMVPCClusterStatus defines the observed state of IBMVPCCluster.
type IBMVPCClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// dep: rely on Network instead.
	VPC VPC `json:"vpc,omitempty"`

	// network is the status of the VPC network resources for extended VPC Infrastructure support.
	// +optional
	Network *VPCNetworkStatus `json:"network,omitempty"`

	// Ready is true when the provider resource is ready.
	// +optional
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// resourceGroup is the status of the cluster's Resource Group for extended VPC Infrastructure support.
	// +optional
	ResourceGroup *ResourceStatus `json:"resourceGroup,omitempty"`

	Subnet      Subnet      `json:"subnet,omitempty"`
	VPCEndpoint VPCEndpoint `json:"vpcEndpoint,omitempty"`

	// ControlPlaneLoadBalancerState is the status of the load balancer.
	// +optional
	ControlPlaneLoadBalancerState VPCLoadBalancerState `json:"controlPlaneLoadBalancerState,omitempty"`

	// Conditions defines current service state of the load balancer.
	// +optional
	Conditions capiv1beta1.Conditions `json:"conditions,omitempty"`
}

// VPCNetworkStatus provides details on the status of VPC network resources for extended VPC Infrastructure support.
type VPCNetworkStatus struct {
	// resourceGroup references the Resource Group for Network resources for the cluster.
	// This can be the same or unique from the cluster's Resource Group.
	// +optional
	ResourceGroup *ResourceStatus `json:"resourceGroup,omitempty"`

	// vpc references the status of the IBM Cloud VPC as part of the extended VPC Infrastructure support.
	// +optional
	VPC *ResourceStatus `json:"vpc,omitempty"`
}

// VPC holds the VPC information.
type VPC struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=ibmvpcclusters,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this IBMVPCCluster belongs"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready for IBM VPC instances"

// IBMVPCCluster is the Schema for the ibmvpcclusters API.
type IBMVPCCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IBMVPCClusterSpec   `json:"spec,omitempty"`
	Status IBMVPCClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IBMVPCClusterList contains a list of IBMVPCCluster.
type IBMVPCClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBMVPCCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IBMVPCCluster{}, &IBMVPCClusterList{})
}

// GetConditions returns the observations of the operational state of the IBMVPCCluster resource.
func (r *IBMVPCCluster) GetConditions() capiv1beta1.Conditions {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the IBMVPCCluster to the predescribed clusterv1.Conditions.
func (r *IBMVPCCluster) SetConditions(conditions capiv1beta1.Conditions) {
	r.Status.Conditions = conditions
}
