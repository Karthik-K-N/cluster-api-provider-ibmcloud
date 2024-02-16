/*
Copyright 2021 The Kubernetes Authors.

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

package scope

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	"github.com/IBM-Cloud/power-go-client/ibmpisession"
	"github.com/IBM-Cloud/power-go-client/power/models"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/ibm-cos-sdk-go/aws/awserr"
	"github.com/IBM/ibm-cos-sdk-go/service/s3"
	tgapiv1 "github.com/IBM/networking-go-sdk/transitgatewayapisv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"

	"k8s.io/klog/v2/klogr"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/authenticator"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/cos"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/globalcatalog"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/powervs"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/resourcecontroller"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/transitgateway"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/utils"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/vpc"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/endpoints"
	genUtil "sigs.k8s.io/cluster-api-provider-ibmcloud/util"
)

const (
	// DEBUGLEVEL indicates the debug level of the logs.
	DEBUGLEVEL = 5
)

const (
	// TODO(karthik-k-n)(Doubt): should this be fetched using global catalogs or hardcode like this?

	// powerVSResourcePlanID is Power VS power-iaas plan id, can be retrieved using ibmcloud cli
	// ibmcloud catalog service power-iaas.
	powerVSResourcePlanID = "f165dd34-3a40-423b-9d95-e90a23f724dd"

	// cosResourceID is IBM COS service id, can be retrieved using ibmcloud cli
	// ibmcloud catalog service cloud-object-storage.
	cosResourceID = "dff97f5c-bc5e-4455-b470-411c3edbe49c"

	// powerVSResourcePlanID is IBM COS plan id, can be retrieved using ibmcloud cli
	// ibmcloud catalog service cloud-object-storage.
	cosResourcePlanID = "1e4e33e4-cfa6-4f12-9016-be594a6d5f87"
)

// ResourceType describes IBM Cloud resource name.
type ResourceType string

var (
	// ServiceInstance is Power VS service instance resource.
	ServiceInstance = ResourceType("serviceInstance")
	// Network is Power VS network resource.
	Network = ResourceType("network")
	// DHCPServer is Power VS DHCP server.
	DHCPServer = ResourceType("dhcpServer")
	// LoadBalancer VPC loadBalancer resource.
	LoadBalancer = ResourceType("loadBalancer")
	// TransitGateway is transit gateway resource.
	TransitGateway = ResourceType("transitGateway")
	// VPC is Power VS network resource.
	VPC = ResourceType("vpc")
	// Subnet VPC subnet resource.
	Subnet = ResourceType("subnet")
	// COSInstance is IBM COS instance resource.
	COSInstance = ResourceType("cosInstance")
)

// PowerVSClusterScopeParams defines the input parameters used to create a new PowerVSClusterScope.
type PowerVSClusterScopeParams struct {
	Client            client.Client
	Logger            logr.Logger
	Cluster           *capiv1beta1.Cluster
	IBMPowerVSCluster *infrav1beta2.IBMPowerVSCluster
	ServiceEndpoint   []endpoints.ServiceEndpoint
}

// PowerVSClusterScope defines a scope defined around a Power VS Cluster.
type PowerVSClusterScope struct {
	logr.Logger
	Client      client.Client
	patchHelper *patch.Helper
	session     *ibmpisession.IBMPISession

	IBMPowerVSClient     powervs.PowerVS
	IBMVPCClient         vpc.Vpc
	TransitGatewayClient transitgateway.TransitGateway
	ResourceClient       resourcecontroller.ResourceController
	CatalogClient        globalcatalog.GlobalCatalog
	COSClient            cos.Cos

	Cluster           *capiv1beta1.Cluster
	IBMPowerVSCluster *infrav1beta2.IBMPowerVSCluster
	ServiceEndpoint   []endpoints.ServiceEndpoint
}

// ClusterObject represents a IBMPowerVS cluster object.
type ClusterObject interface {
	conditions.Setter
}

// NewPowerVSClusterScope creates a new PowerVSClusterScope from the supplied parameters.
func NewPowerVSClusterScope(params PowerVSClusterScopeParams) (*PowerVSClusterScope, error) { //nolint:gocyclo
	if params.Client == nil {
		err := errors.New("error failed to generate new scope from nil Client")
		return nil, err
	}
	if params.Cluster == nil {
		err := errors.New("error failed to generate new scope from nil Cluster")
		return nil, err
	}
	if params.IBMPowerVSCluster == nil {
		err := errors.New("error failed to generate new scope from nil IBMPowerVSCluster")
		return nil, err
	}
	if params.Logger == (logr.Logger{}) {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.IBMPowerVSCluster, params.Client)
	if err != nil {
		err = fmt.Errorf("error failed to init patch helper: %w", err)
		return nil, err
	}

	options := powervs.ServiceOptions{
		IBMPIOptions: &ibmpisession.IBMPIOptions{
			Debug: params.Logger.V(DEBUGLEVEL).Enabled(),
		},
	}

	if params.IBMPowerVSCluster.Spec.ServiceInstanceID != "" {
		rc, err := resourcecontroller.NewService(resourcecontroller.ServiceOptions{})
		if err != nil {
			return nil, err
		}

		// Fetch the resource controller endpoint.
		if rcEndpoint := endpoints.FetchRCEndpoint(params.ServiceEndpoint); rcEndpoint != "" {
			if err := rc.SetServiceURL(rcEndpoint); err != nil {
				return nil, fmt.Errorf("failed to set resource controller endpoint: %w", err)
			}
		}

		res, _, err := rc.GetResourceInstance(
			&resourcecontrollerv2.GetResourceInstanceOptions{
				ID: core.StringPtr(params.IBMPowerVSCluster.Spec.ServiceInstanceID),
			})
		if err != nil {
			err = fmt.Errorf("failed to get resource instance: %w", err)
			return nil, err
		}
		options.Zone = *res.RegionID
		options.CloudInstanceID = params.IBMPowerVSCluster.Spec.ServiceInstanceID
	} else {
		options.Zone = *params.IBMPowerVSCluster.Spec.Zone
	}

	// TODO(karhtik-k-n): may be optimize NewService to use the session created here
	powerVSClient, err := powervs.NewService(options)
	if err != nil {
		return nil, fmt.Errorf("error failed to create power vs client %w", err)
	}

	auth, err := authenticator.GetAuthenticator()
	if err != nil {
		return nil, fmt.Errorf("error failed to create authenticator %w", err)
	}
	account, err := utils.GetAccount(auth)
	if err != nil {
		return nil, fmt.Errorf("error failed to get account details %w", err)
	}
	// TODO(Karthik-k-n): Handle dubug and URL options.
	sessionOptions := &ibmpisession.IBMPIOptions{
		Authenticator: auth,
		UserAccount:   account,
		Zone:          options.Zone,
	}
	session, err := ibmpisession.NewIBMPISession(sessionOptions)
	if err != nil {
		return nil, fmt.Errorf("error failed to get power vs session %w", err)
	}

	if !genUtil.CreateInfra(*params.IBMPowerVSCluster) {
		return &PowerVSClusterScope{
			session:           session,
			Logger:            params.Logger,
			Client:            params.Client,
			patchHelper:       helper,
			Cluster:           params.Cluster,
			IBMPowerVSCluster: params.IBMPowerVSCluster,
			ServiceEndpoint:   params.ServiceEndpoint,
			IBMPowerVSClient:  powerVSClient,
		}, nil
	}

	if params.IBMPowerVSCluster.Spec.VPC == nil || params.IBMPowerVSCluster.Spec.VPC.Region == nil {
		return nil, fmt.Errorf("error failed to generate vpc client as VPC info is nil")
	}

	if params.Logger.V(DEBUGLEVEL).Enabled() {
		core.SetLoggingLevel(core.LevelDebug)
	}

	svcEndpoint := endpoints.FetchVPCEndpoint(*params.IBMPowerVSCluster.Spec.VPC.Region, params.ServiceEndpoint)
	vpcClient, err := vpc.NewService(svcEndpoint)
	if err != nil {
		return nil, fmt.Errorf("error ailed to create IBM VPC client: %w", err)
	}

	tgClient, err := transitgateway.NewService()
	if err != nil {
		return nil, fmt.Errorf("error failed to create tranist gateway client: %w", err)
	}

	// TODO(karthik-k-n): consider passing auth in options to resource controller
	resourceClient, err := resourcecontroller.NewService(resourcecontroller.ServiceOptions{})
	if err != nil {
		return nil, fmt.Errorf("error failed to create resource client: %w", err)
	}

	catalogClient, err := globalcatalog.NewService(globalcatalog.ServiceOptions{})
	if err != nil {
		return nil, fmt.Errorf("error failed to create catalog client: %w", err)
	}

	clusterScope := &PowerVSClusterScope{
		session:              session,
		Logger:               params.Logger,
		Client:               params.Client,
		patchHelper:          helper,
		Cluster:              params.Cluster,
		IBMPowerVSCluster:    params.IBMPowerVSCluster,
		ServiceEndpoint:      params.ServiceEndpoint,
		IBMPowerVSClient:     powerVSClient,
		IBMVPCClient:         vpcClient,
		TransitGatewayClient: tgClient,
		ResourceClient:       resourceClient,
		CatalogClient:        catalogClient,
	}
	return clusterScope, nil
}

// PatchObject persists the cluster configuration and status.
func (s *PowerVSClusterScope) PatchObject() error {
	return s.patchHelper.Patch(context.TODO(), s.IBMPowerVSCluster)
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *PowerVSClusterScope) Close() error {
	return s.PatchObject()
}

// Name returns the CAPI cluster name.
func (s *PowerVSClusterScope) Name() string {
	return s.Cluster.Name
}

// Zone returns the cluster zone.
func (s *PowerVSClusterScope) Zone() *string {
	return s.IBMPowerVSCluster.Spec.Zone
}

// ResourceGroup returns the cluster resource group.
func (s *PowerVSClusterScope) ResourceGroup() *string {
	return s.IBMPowerVSCluster.Spec.ResourceGroup
}

// InfraCluster returns the IBMPowerVS infrastructure cluster object.
func (s *PowerVSClusterScope) InfraCluster() ClusterObject {
	return s.IBMPowerVSCluster
}

// APIServerPort returns the APIServerPort to use when creating the ControlPlaneEndpoint.
func (s *PowerVSClusterScope) APIServerPort() int32 {
	if s.Cluster.Spec.ClusterNetwork != nil && s.Cluster.Spec.ClusterNetwork.APIServerPort != nil {
		return *s.Cluster.Spec.ClusterNetwork.APIServerPort
	}
	return 6443
}

// ServiceInstance returns the cluster ServiceInstance.
func (s *PowerVSClusterScope) ServiceInstance() *infrav1beta2.IBMPowerVSResourceReference {
	return s.IBMPowerVSCluster.Spec.ServiceInstance
}

// GetServiceInstanceID get the service instance id.
func (s *PowerVSClusterScope) GetServiceInstanceID() string {
	// support the existing worklfow
	if s.IBMPowerVSCluster.Spec.ServiceInstanceID != "" {
		return s.IBMPowerVSCluster.Spec.ServiceInstanceID
	}
	if s.IBMPowerVSCluster.Spec.ServiceInstance != nil && s.IBMPowerVSCluster.Spec.ServiceInstance.ID != nil {
		return *s.IBMPowerVSCluster.Spec.ServiceInstance.ID
	}
	if s.IBMPowerVSCluster.Status.ServiceInstance != nil && s.IBMPowerVSCluster.Status.ServiceInstance.ID != nil {
		return *s.IBMPowerVSCluster.Status.ServiceInstance.ID
	}
	return ""
}

// TODO: Can we use generic here.

// SetStatus set the IBMPowerVSCluster status.
func (s *PowerVSClusterScope) SetStatus(resourceType ResourceType, resource infrav1beta2.ResourceReference) {
	switch resourceType {
	case ServiceInstance:
		if s.IBMPowerVSCluster.Status.ServiceInstance == nil {
			s.IBMPowerVSCluster.Status.ServiceInstance = &resource
			return
		}
		s.IBMPowerVSCluster.Status.ServiceInstance.Set(resource)
	case Network:
		if s.IBMPowerVSCluster.Status.Network == nil {
			s.IBMPowerVSCluster.Status.Network = &resource
			return
		}
		s.IBMPowerVSCluster.Status.Network.Set(resource)
	case VPC:
		if s.IBMPowerVSCluster.Status.VPC == nil {
			s.IBMPowerVSCluster.Status.VPC = &resource
			return
		}
		s.IBMPowerVSCluster.Status.VPC.Set(resource)
	case TransitGateway:
		if s.IBMPowerVSCluster.Status.TransitGateway == nil {
			s.IBMPowerVSCluster.Status.TransitGateway = &resource
			return
		}
		s.IBMPowerVSCluster.Status.TransitGateway.Set(resource)
	case DHCPServer:
		if s.IBMPowerVSCluster.Status.DHCPServer == nil {
			s.IBMPowerVSCluster.Status.DHCPServer = &resource
			return
		}
		s.IBMPowerVSCluster.Status.DHCPServer.Set(resource)
	case COSInstance:
		if s.IBMPowerVSCluster.Status.COSInstance == nil {
			s.IBMPowerVSCluster.Status.COSInstance = &resource
			return
		}
		s.IBMPowerVSCluster.Status.COSInstance.Set(resource)
	}
}

// Network returns the cluster Network.
func (s *PowerVSClusterScope) Network() infrav1beta2.IBMPowerVSResourceReference {
	return s.IBMPowerVSCluster.Spec.Network
}

// SetNetworkStatus set the network status.
func (s *PowerVSClusterScope) SetNetworkStatus(resource infrav1beta2.ResourceReference) {
	if s.IBMPowerVSCluster.Status.Network == nil {
		s.IBMPowerVSCluster.Status.Network = &resource
	}
	s.IBMPowerVSCluster.Status.Network.Set(resource)
}

// GetDHCPServerID returns the DHCP id.
func (s *PowerVSClusterScope) GetDHCPServerID() *string {
	if s.IBMPowerVSCluster.Status.DHCPServer != nil {
		return s.IBMPowerVSCluster.Status.DHCPServer.ID
	}
	return nil
}

// VPC returns the cluster VPC information.
func (s *PowerVSClusterScope) VPC() *infrav1beta2.VPCResourceReference {
	return s.IBMPowerVSCluster.Spec.VPC
}

// GetVPCID returns the VPC id.
func (s *PowerVSClusterScope) GetVPCID() *string {
	if s.IBMPowerVSCluster.Spec.VPC != nil && s.IBMPowerVSCluster.Spec.VPC.ID != nil {
		return s.IBMPowerVSCluster.Spec.VPC.ID
	}
	if s.IBMPowerVSCluster.Status.VPC != nil {
		return s.IBMPowerVSCluster.Status.VPC.ID
	}
	return nil
}

// GetVPCSubnetID returns the VPC subnet id.
func (s *PowerVSClusterScope) GetVPCSubnetID(subnet infrav1beta2.Subnet) *string {
	if subnet.ID != nil {
		return subnet.ID
	}
	if subnet.Name == nil || s.IBMPowerVSCluster.Status.VPCSubnet == nil {
		return nil
	}
	if val, ok := s.IBMPowerVSCluster.Status.VPCSubnet[*subnet.Name]; ok {
		return val.ID
	}
	return nil
}

// GetVPCSubnetIDs returns all the VPC subnet ids.
func (s *PowerVSClusterScope) GetVPCSubnetIDs() []*string {
	if s.IBMPowerVSCluster.Status.VPCSubnet == nil {
		return nil
	}
	subnets := []*string{}
	for _, subnet := range s.IBMPowerVSCluster.Status.VPCSubnet {
		subnets = append(subnets, subnet.ID)
	}
	return subnets
}

// SetVPCSubnetID set the VPC subnet id.
func (s *PowerVSClusterScope) SetVPCSubnetID(name string, resource infrav1beta2.ResourceReference) {
	if s.IBMPowerVSCluster.Status.VPCSubnet == nil {
		s.IBMPowerVSCluster.Status.VPCSubnet = make(map[string]infrav1beta2.ResourceReference)
	}
	if val, ok := s.IBMPowerVSCluster.Status.VPCSubnet[name]; ok {
		if val.ControllerCreated != nil && *val.ControllerCreated {
			resource.ControllerCreated = val.ControllerCreated
		}
	}
	s.IBMPowerVSCluster.Status.VPCSubnet[name] = resource
}

// TransitGateway returns the cluster Transit Gateway information.
func (s *PowerVSClusterScope) TransitGateway() *infrav1beta2.TransitGateway {
	return s.IBMPowerVSCluster.Spec.TransitGateway
}

// GetTransitGatewayID returns the transit gateway id.
func (s *PowerVSClusterScope) GetTransitGatewayID() *string {
	if s.IBMPowerVSCluster.Spec.TransitGateway != nil && s.IBMPowerVSCluster.Spec.TransitGateway.ID != nil {
		return s.IBMPowerVSCluster.Spec.TransitGateway.ID
	}
	if s.IBMPowerVSCluster.Status.TransitGateway != nil {
		return s.IBMPowerVSCluster.Status.TransitGateway.ID
	}
	return nil
}

// PublicLoadBalancer returns the cluster public loadBalancer information.
func (s *PowerVSClusterScope) PublicLoadBalancer() *infrav1beta2.VPCLoadBalancerSpec {
	if s.IBMPowerVSCluster.Spec.LoadBalancers == nil {
		return nil
	}
	for _, lb := range s.IBMPowerVSCluster.Spec.LoadBalancers {
		if lb.Public {
			return &lb
		}
	}
	return nil
}

// SetLoadBalancerStatus set the loadBalancer id.
func (s *PowerVSClusterScope) SetLoadBalancerStatus(name string, loadBalancer infrav1beta2.VPCLoadBalancerStatus) {
	if s.IBMPowerVSCluster.Status.LoadBalancers == nil {
		s.IBMPowerVSCluster.Status.LoadBalancers = make(map[string]infrav1beta2.VPCLoadBalancerStatus)
	}
	if val, ok := s.IBMPowerVSCluster.Status.LoadBalancers[name]; ok {
		if val.ControllerCreated != nil && *val.ControllerCreated {
			loadBalancer.ControllerCreated = val.ControllerCreated
		}
	}
	s.IBMPowerVSCluster.Status.LoadBalancers[name] = loadBalancer
}

// GetLoadBalancerID returns the loadBalancer.
func (s *PowerVSClusterScope) GetLoadBalancerID(loadBalancer infrav1beta2.VPCLoadBalancerSpec) *string {
	if loadBalancer.Name == "" || s.IBMPowerVSCluster.Status.LoadBalancers == nil {
		return nil
	}
	if val, ok := s.IBMPowerVSCluster.Status.LoadBalancers[loadBalancer.Name]; ok {
		return val.ID
	}
	return nil
}

// GetLoadBalancerState will get the state for the load balancer.
func (s *PowerVSClusterScope) GetLoadBalancerState(name string) *infrav1beta2.VPCLoadBalancerState {
	if s.IBMPowerVSCluster.Status.LoadBalancers == nil {
		return nil
	}
	if val, ok := s.IBMPowerVSCluster.Status.LoadBalancers[name]; ok {
		return &val.State
	}
	return nil
}

// GetLoadBalancerHost will return the hostname of load balancer.
func (s *PowerVSClusterScope) GetLoadBalancerHost(name string) *string {
	if s.IBMPowerVSCluster.Status.LoadBalancers == nil {
		return nil
	}
	if val, ok := s.IBMPowerVSCluster.Status.LoadBalancers[name]; ok {
		return val.Hostname
	}
	return nil
}

// SetCOSBucketStatus set the COS bucket id.
func (s *PowerVSClusterScope) SetCOSBucketStatus(cosInstanceID string, controllerCreated bool) {
	if s.IBMPowerVSCluster.Status.COSInstance == nil {
		s.IBMPowerVSCluster.Status.COSInstance = &infrav1beta2.ResourceReference{
			ID:                &cosInstanceID,
			ControllerCreated: &controllerCreated,
		}
	} else {
		// set the controllerCreated only if its not set
		if !*s.IBMPowerVSCluster.Status.COSInstance.ControllerCreated {
			s.IBMPowerVSCluster.Status.COSInstance.ControllerCreated = &controllerCreated
		}
		s.IBMPowerVSCluster.Status.COSInstance.ID = &cosInstanceID
	}
}

// ReconcileServiceInstance reconciles service instance.
func (s *PowerVSClusterScope) ReconcileServiceInstance() error {
	serviceInstanceID := s.GetServiceInstanceID()
	if serviceInstanceID != "" {
		s.Info("Service instance id is set", "id", serviceInstanceID)
		serviceInstance, _, err := s.ResourceClient.GetResourceInstance(&resourcecontrollerv2.GetResourceInstanceOptions{
			ID: &serviceInstanceID,
		})
		if err != nil {
			return err
		}
		if serviceInstance == nil {
			return fmt.Errorf("error failed to get service instance with id %s", serviceInstanceID)
		}
		if *serviceInstance.State != string(infrav1beta2.ServiceInstanceStateActive) {
			return fmt.Errorf("service instance not in active state, current state: %s", *serviceInstance.State)
		}
		s.Info("Found service instance and its in active state", "id", serviceInstanceID)
		return nil
	}

	// check service instance exist in cloud
	serviceInstanceID, err := s.checkServiceInstance()
	if err != nil {
		return err
	}
	if serviceInstanceID != "" {
		s.SetStatus(ServiceInstance, infrav1beta2.ResourceReference{ID: &serviceInstanceID, ControllerCreated: pointer.Bool(false)})
		return nil
	}

	// create Service Instance
	serviceInstance, err := s.createServiceInstance()
	if err != nil {
		return err
	}
	s.SetStatus(ServiceInstance, infrav1beta2.ResourceReference{ID: serviceInstance.GUID, ControllerCreated: pointer.Bool(true)})
	return nil
}

// checkServiceInstance checks service instance exist in cloud.
func (s *PowerVSClusterScope) checkServiceInstance() (string, error) {
	s.Info("Checking for service instance in cloud")
	serviceInstance, err := s.getServiceInstance()
	if err != nil {
		s.Error(err, "failed to get service instance")
		return "", err
	}
	if serviceInstance == nil {
		s.Info("Not able to find service instance", "service instance", s.IBMPowerVSCluster.Spec.ServiceInstance)
		return "", nil
	}
	if *serviceInstance.State != string(infrav1beta2.ServiceInstanceStateActive) {
		s.Info("Service instance not in active state", "service instance", s.IBMPowerVSCluster.Spec.ServiceInstance, "state", *serviceInstance.State)
		return "", fmt.Errorf("service instance not in active state, current state: %s", *serviceInstance.State)
	}
	s.Info("Service instance found and its in active state", "id", *serviceInstance.GUID)
	return *serviceInstance.GUID, nil
}

// getServiceInstance return resource instance.
func (s *PowerVSClusterScope) getServiceInstance() (*resourcecontrollerv2.ResourceInstance, error) {
	//TODO: Support regular expression
	return s.ResourceClient.GetServiceInstance("", *s.GetServiceName(ServiceInstance))
}

// createServiceInstance creates the service instance.
func (s *PowerVSClusterScope) createServiceInstance() (*resourcecontrollerv2.ResourceInstance, error) {
	resourceGroupID, err := s.getResourceGroupID()
	if err != nil {
		s.Error(err, "failed to create service instance, failed to getch resource group id")
		return nil, fmt.Errorf("error getting id for resource group %s, %w", *s.ResourceGroup(), err)
	}

	// TODO: Do we need to fetch or hardcode
	// _, servicePlanID, err := s.CatalogClient.GetServiceInfo(powerVSService, powerVSServicePlan)
	// if err != nil {
	//	return nil, fmt.Errorf("error retrieving id info for powervs service %w", err)
	//}

	// create service instance
	s.Info("Creating new service instance", "name", s.GetServiceName(ServiceInstance))
	serviceInstance, _, err := s.ResourceClient.CreateResourceInstance(&resourcecontrollerv2.CreateResourceInstanceOptions{
		Name:           s.GetServiceName(ServiceInstance),
		Target:         s.Zone(),
		ResourceGroup:  &resourceGroupID,
		ResourcePlanID: pointer.String(powerVSResourcePlanID),
	})
	if err != nil {
		return nil, err
	}
	s.Info("Created new service instance")
	return serviceInstance, nil
}

// ReconcileNetwork reconciles network.
func (s *PowerVSClusterScope) ReconcileNetwork() error {
	// if DHCP server id is set means the server is already created
	if s.GetDHCPServerID() != nil {
		s.Info("DHCP server id is set")
		err := s.checkDHCPServerStatus()
		if err != nil {
			return err
		}
		return nil
		//	TODO(karthik-k-n): If needed set dhcp status here
	}
	// check network exist in cloud
	networkID, err := s.checkNetwork()
	if err != nil {
		return err
	}
	if networkID != "" {
		s.Info("Found network", "id", networkID)
		s.SetStatus(Network, infrav1beta2.ResourceReference{ID: &networkID, ControllerCreated: pointer.Bool(false)})
		return nil
	}

	s.Info("Creating DHCP server")
	dhcpServer, err := s.createDHCPServer()
	if err != nil {
		s.Error(err, "Error creating DHCP server")
		return err
	}
	if dhcpServer != nil {
		s.Info("Created DHCP Server", "id", *dhcpServer)
		s.SetStatus(DHCPServer, infrav1beta2.ResourceReference{ID: dhcpServer, ControllerCreated: pointer.Bool(true)})
		return nil
	}

	err = s.checkDHCPServerStatus()
	if err != nil {
		return err
	}
	return nil
}

// checkNetwork checks the network exist in cloud.
func (s *PowerVSClusterScope) checkNetwork() (string, error) {
	networkID, err := s.getNetwork()
	if err != nil {
		s.Error(err, "failed to get network")
		return "", err
	}
	if networkID == nil {
		s.Info("Not able to find network", "network", s.IBMPowerVSCluster.Spec.Network)
		return "", nil
	}
	return *networkID, nil
}

func (s *PowerVSClusterScope) getNetwork() (*string, error) {
	if s.IBMPowerVSCluster.Spec.Network.ID != nil {
		network, err := s.IBMPowerVSClient.GetNetworkByID(*s.IBMPowerVSCluster.Spec.Network.ID)
		if err != nil {
			return nil, err
		}
		return network.NetworkID, nil
	}
	network, err := s.IBMPowerVSClient.GetNetworkByName(*s.GetServiceName(Network))
	if err != nil {
		return nil, err
	}
	if network == nil {
		s.Info("network does not exist", "name", *s.GetServiceName(Network))
		return nil, nil
	}
	return network.NetworkID, nil
	//TODO: Support regular expression
}

// checkDHCPServerStatus checks the DHCP server status.
func (s *PowerVSClusterScope) checkDHCPServerStatus() error {
	dhcpID := *s.GetDHCPServerID()
	dhcpServer, err := s.IBMPowerVSClient.GetDHCPServer(dhcpID)
	if err != nil {
		return err
	}
	if dhcpServer == nil {
		return fmt.Errorf("error failed to get dchp server")
	}
	if *dhcpServer.Status != "ACTIVE" {
		return fmt.Errorf("error dhcp server state is not active, current state %s", *dhcpServer.Status)
	}
	s.Info("DHCP server is found and its in active state")
	return nil
}

// createDHCPServer creates the DHCP server.
func (s *PowerVSClusterScope) createDHCPServer() (*string, error) {
	dhcpServer, err := s.IBMPowerVSClient.CreateDHCPServer(&models.DHCPServerCreate{
		DNSServer: pointer.String("1.1.1.1"),
		Name:      s.GetServiceName(DHCPServer),
	})
	if err != nil {
		return nil, err
	}
	if dhcpServer == nil {
		return nil, fmt.Errorf("created dhcp server is nil")
	}
	if dhcpServer.Network == nil {
		return nil, fmt.Errorf("created dhcp server network is nil")
	}

	s.Info("DHCP Server network details", "details", *dhcpServer.Network)
	s.SetStatus(Network, infrav1beta2.ResourceReference{ID: dhcpServer.Network.ID, ControllerCreated: pointer.Bool(true)})
	return dhcpServer.ID, nil
}

// ReconcileVPC reconciles VPC.
func (s *PowerVSClusterScope) ReconcileVPC() error {
	// if VPC server id is set means the VPC is already created
	vpcID := s.GetVPCID()
	if vpcID != nil {
		s.Info("VPC id is set", "id", vpcID)
		vpcDetails, _, err := s.IBMVPCClient.GetVPC(&vpcv1.GetVPCOptions{
			ID: vpcID,
		})
		if err != nil {
			return err
		}
		if vpcDetails == nil {
			return fmt.Errorf("error failed to get vpc with id %s", *vpcID)
		}
		s.Info("Found VPC with provided id")
		// TODO(karthik-k-n): Set status here as well
		return nil
	}

	// check vpc exist in cloud
	id, err := s.checkVPC()
	if err != nil {
		return err
	}
	if id != "" {
		s.SetStatus(VPC, infrav1beta2.ResourceReference{ID: &id, ControllerCreated: pointer.Bool(false)})
		return nil
	}

	// TODO(karthik-k-n): create a generic cluster scope/service and implement common vpc logics, which can be consumed by both vpc and powervs

	// create VPC
	s.Info("Creating a VPC")
	vpcDetails, err := s.createVPC()
	if err != nil {
		return err
	}
	s.Info("Successfully create VPC")
	s.SetStatus(VPC, infrav1beta2.ResourceReference{ID: vpcDetails, ControllerCreated: pointer.Bool(true)})
	return nil
}

// checkVPC checks VPC exist in cloud.
func (s *PowerVSClusterScope) checkVPC() (string, error) {
	vpcDetails, err := s.getVPC()
	if err != nil {
		s.Error(err, "failed to get vpc")
		return "", err
	}
	if vpcDetails == nil {
		s.Info("Not able to find vpc", "vpc", s.IBMPowerVSCluster.Spec.VPC)
		return "", nil
	}
	s.Info("VPC found", "id", *vpcDetails.ID)
	return *vpcDetails.ID, nil
}

func (s *PowerVSClusterScope) getVPC() (*vpcv1.VPC, error) {
	vpcDetails, err := s.IBMVPCClient.GetVPCByName(*s.GetServiceName(VPC))
	if err != nil {
		return nil, err
	}
	return vpcDetails, nil
	//TODO: Support regular expression
}

// createVPC creates VPC.
func (s *PowerVSClusterScope) createVPC() (*string, error) {
	resourceGroupID, err := s.getResourceGroupID()
	if err != nil {
		return nil, fmt.Errorf("failed to create vpc, error getting id for resource group %s, %w", *s.ResourceGroup(), err)
	}
	addressPrefixManagement := "auto"
	vpcOption := &vpcv1.CreateVPCOptions{
		ResourceGroup:           &vpcv1.ResourceGroupIdentity{ID: &resourceGroupID},
		Name:                    s.GetServiceName(VPC),
		AddressPrefixManagement: &addressPrefixManagement,
	}
	vpcDetails, _, err := s.IBMVPCClient.CreateVPC(vpcOption)
	if err != nil {
		return nil, err
	}

	// set security group for vpc
	options := &vpcv1.CreateSecurityGroupRuleOptions{}
	options.SetSecurityGroupID(*vpcDetails.DefaultSecurityGroup.ID)
	options.SetSecurityGroupRulePrototype(&vpcv1.SecurityGroupRulePrototype{
		Direction: core.StringPtr("inbound"),
		Protocol:  core.StringPtr("tcp"),
		IPVersion: core.StringPtr("ipv4"),
		PortMin:   core.Int64Ptr(int64(s.APIServerPort())),
		PortMax:   core.Int64Ptr(int64(s.APIServerPort())),
	})
	_, _, err = s.IBMVPCClient.CreateSecurityGroupRule(options)
	if err != nil {
		return nil, err
	}
	return vpcDetails.ID, nil
}

// ReconcileVPCSubnet reconciles VPC subnet.
func (s *PowerVSClusterScope) ReconcileVPCSubnet() error {
	//TODO: Should we handle a case where there are no subnet specified
	for _, subnet := range s.IBMPowerVSCluster.Spec.VPCSubnets {
		if subnet.Name == nil {
			subnet.Name = pointer.String(fmt.Sprintf("%s-vpcsubnet", s.InfraCluster().GetName()))
		}
		s.Info("Reconciling vpc subnet", "subnet", subnet.Name)
		subnetID := s.GetVPCSubnetID(subnet)
		if subnetID != nil {
			subnetDetails, _, err := s.IBMVPCClient.GetSubnet(&vpcv1.GetSubnetOptions{
				ID: subnetID,
			})
			if err != nil {
				return err
			}
			if subnetDetails == nil {
				return fmt.Errorf("error failed to get vpc subnet with id %s", *subnetID)
			}
			// check for next subnet
			continue
		}

		// check VPC subnet exist in cloud
		vpcSubnetID, err := s.checkVPCSubnet(subnet)
		if err != nil {
			s.Error(err, "error checking vpc subnet")
			return err
		}
		if vpcSubnetID != "" {
			s.Info("found vpc subnet", "id", vpcSubnetID)
			s.SetVPCSubnetID(*subnet.Name, infrav1beta2.ResourceReference{ID: &vpcSubnetID, ControllerCreated: pointer.Bool(false)})
			// check for next subnet
			continue
		}
		subnetID, err = s.createVPCSubnet(subnet)
		if err != nil {
			s.Error(err, "error creating vpc subnet")
			return err
		}
		if subnetID != nil {
			s.Info("created vpc subnet", "id", subnetID)
			s.SetVPCSubnetID(*subnet.Name, infrav1beta2.ResourceReference{ID: subnetID, ControllerCreated: pointer.Bool(true)})
		}
		// TODO(karthik-k-n)(Doubt): Do we need to create public gateway?
	}
	return nil
}

// checkVPCSubnet checks VPC subnet exist in cloud.
func (s *PowerVSClusterScope) checkVPCSubnet(subnet infrav1beta2.Subnet) (string, error) {
	vpcSubnet, err := s.IBMVPCClient.GetVPCSubnetByName(*subnet.Name)
	if err != nil {
		return "", err
	}
	if vpcSubnet == nil {
		return "", nil
	}
	return *vpcSubnet.ID, nil
}

// createVPCSubnet creates a VPC subnet.
func (s *PowerVSClusterScope) createVPCSubnet(subnet infrav1beta2.Subnet) (*string, error) {
	// TODO(karthik-k-n): consider moving to clusterscope
	// fetch resource group id
	resourceGroupID, err := s.getResourceGroupID()
	if err != nil {
		return nil, fmt.Errorf("error getting id for resource group %s, %w", *s.ResourceGroup(), err)
	}
	var zone string
	if subnet.Zone != nil {
		zone = *subnet.Zone
	} else {
		if s.Zone() == nil {
			return nil, fmt.Errorf("error getting powervs zone")
		}
		region := endpoints.ConstructRegionFromZone(*s.Zone())
		vpcZones, err := genUtil.VPCZonesForPowerVSRegion(region)
		if err != nil {
			return nil, err
		}
		// TODO(karthik-k-n): Decide on using all zones or using one zone
		if len(vpcZones) == 0 {
			return nil, fmt.Errorf("error getting vpc zones error: %v", err)
		}
		zone = vpcZones[0]
	}

	// create subnet
	vpcID := s.GetVPCID()
	cidrBlock, err := s.IBMVPCClient.GetSubnetAddrPrefix(*vpcID, zone)
	if err != nil {
		return nil, err
	}
	ipVersion := "ipv4"

	options := &vpcv1.CreateSubnetOptions{}
	options.SetSubnetPrototype(&vpcv1.SubnetPrototype{
		IPVersion:     &ipVersion,
		Ipv4CIDRBlock: &cidrBlock,
		Name:          subnet.Name,
		VPC: &vpcv1.VPCIdentity{
			ID: vpcID,
		},
		Zone: &vpcv1.ZoneIdentity{
			Name: &zone,
		},
		ResourceGroup: &vpcv1.ResourceGroupIdentity{
			ID: &resourceGroupID,
		},
	})

	subnetDetails, _, err := s.IBMVPCClient.CreateSubnet(options)
	if err != nil {
		return nil, err
	}
	if subnetDetails == nil {
		return nil, fmt.Errorf("create subnet is nil")
	}
	return subnetDetails.ID, nil
}

// ReconcileTransitGateway reconcile transit gateway.
func (s *PowerVSClusterScope) ReconcileTransitGateway() error {
	if s.GetTransitGatewayID() != nil {
		s.Info("TransitGateway id is set", "id", s.GetTransitGatewayID())
		tg, _, err := s.TransitGatewayClient.GetTransitGateway(&tgapiv1.GetTransitGatewayOptions{
			ID: s.GetTransitGatewayID(),
		})
		if err != nil {
			return err
		}
		err = s.checkTransitGatewayStatus(tg.ID)
		if err != nil {
			return err
		}
		return nil
	}

	// check transit gateway exist in cloud
	tgID, err := s.checkTransitGateway()
	if err != nil {
		return err
	}
	if tgID != "" {
		s.SetStatus(TransitGateway, infrav1beta2.ResourceReference{ID: &tgID, ControllerCreated: pointer.Bool(false)})
		return nil
	}
	// create transit gateway
	transitGatewayID, err := s.createTransitGateway()
	if err != nil {
		return err
	}
	if transitGatewayID != nil {
		s.SetStatus(TransitGateway, infrav1beta2.ResourceReference{ID: transitGatewayID, ControllerCreated: pointer.Bool(true)})
		return nil
	}

	// verify that the transit gateway connections are attached
	err = s.checkTransitGatewayStatus(transitGatewayID)
	if err != nil {
		return err
	}
	return nil
}

// checkTransitGateway checks transit gateway exist in cloud.
func (s *PowerVSClusterScope) checkTransitGateway() (string, error) {
	// TODO(karthik-k-n): Support regex
	transitGateway, err := s.TransitGatewayClient.GetTransitGatewayByName(*s.GetServiceName(TransitGateway))
	if err != nil {
		return "", err
	}
	if transitGateway == nil {
		return "", nil
	}
	if err = s.checkTransitGatewayStatus(transitGateway.ID); err != nil {
		return "", err
	}
	return *transitGateway.ID, nil
}

// checkTransitGatewayStatus checks transit gateway status in cloud.
func (s *PowerVSClusterScope) checkTransitGatewayStatus(transitGatewayID *string) error {
	transitGateway, _, err := s.TransitGatewayClient.GetTransitGateway(&tgapiv1.GetTransitGatewayOptions{
		ID: transitGatewayID,
	})
	if err != nil {
		return err
	}
	if transitGateway == nil {
		return fmt.Errorf("tranist gateway is nil")
	}
	if *transitGateway.Status != "available" {
		return fmt.Errorf("error tranist gateway %s not in available status, current status: %s", *transitGatewayID, *transitGateway.Status)
	}

	tgConnections, _, err := s.TransitGatewayClient.ListTransitGatewayConnections(&tgapiv1.ListTransitGatewayConnectionsOptions{
		TransitGatewayID: transitGateway.ID,
	})
	if err != nil {
		return fmt.Errorf("error listing transit gateway connections: %w", err)
	}

	for _, conn := range tgConnections.Connections {
		if *conn.NetworkType == "vpc" && *conn.Status != "attached" {
			return fmt.Errorf("error vpc connection not attached to transit gateway, current status: %s", *conn.Status)
		}
		if *conn.NetworkType == "power_virtual_server" && *conn.Status != "attached" {
			return fmt.Errorf("error powervs connection not attached to transit gateway, current status: %s", *conn.Status)
		}
	}
	return nil
}

// createTransitGateway create transit gateway.
func (s *PowerVSClusterScope) createTransitGateway() (*string, error) {
	// TODO(karthik-k-n): Verify that the supplied zone supports PER
	// TODO(karthik-k-n): consider moving to clusterscope
	// fetch resource group id
	resourceGroupID, err := s.getResourceGroupID()
	if err != nil {
		return nil, fmt.Errorf("error getting id for resource group %s, %w", *s.ResourceGroup(), err)
	}

	vpcRegion := s.getVPCRegion()
	if vpcRegion == nil {
		return nil, fmt.Errorf("failed to get vpc region")
	}
	tg, _, err := s.TransitGatewayClient.CreateTransitGateway(&tgapiv1.CreateTransitGatewayOptions{
		Location:      vpcRegion,
		Name:          s.GetServiceName(TransitGateway),
		Global:        pointer.Bool(true),
		ResourceGroup: &tgapiv1.ResourceGroupIdentity{ID: pointer.String(resourceGroupID)},
	})
	if err != nil {
		return nil, fmt.Errorf("error creating transit gateway: %w", err)
	}

	vpcCRN, err := s.fetchVPCCRN()
	if err != nil {
		return nil, fmt.Errorf("error failed to fetch VPC CRN: %w", err)
	}

	_, _, err = s.TransitGatewayClient.CreateTransitGatewayConnection(&tgapiv1.CreateTransitGatewayConnectionOptions{
		TransitGatewayID: tg.ID,
		NetworkType:      pointer.String("vpc"),
		NetworkID:        vpcCRN,
		Name:             pointer.String(fmt.Sprintf("%s-vpc-con", *s.GetServiceName(TransitGateway))),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating vpc connection in transit gateway: %w", err)
	}

	pvsServiceInstanceCRN, err := s.fetchPowerVSServiceInstanceCRN()
	if err != nil {
		return nil, fmt.Errorf("error failed to fetch powervs service instance CRN: %w", err)
	}

	_, _, err = s.TransitGatewayClient.CreateTransitGatewayConnection(&tgapiv1.CreateTransitGatewayConnectionOptions{
		TransitGatewayID: tg.ID,
		NetworkType:      pointer.String("power_virtual_server"),
		NetworkID:        pvsServiceInstanceCRN,
		Name:             pointer.String(fmt.Sprintf("%s-pvs-con", *s.GetServiceName(TransitGateway))),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating powervs connection in transit gateway: %w", err)
	}
	return tg.ID, nil
}

// ReconcileLoadBalancer reconcile loadBalancer.
func (s *PowerVSClusterScope) ReconcileLoadBalancer() error {
	for index, loadBalancer := range s.IBMPowerVSCluster.Spec.LoadBalancers {
		if loadBalancer.Name == "" {
			loadBalancer.Name = fmt.Sprintf("%s-loadbalancer-%d", s.InfraCluster().GetName(), index)
		}
		loadBalancerID := s.GetLoadBalancerID(loadBalancer)
		if loadBalancerID != nil {
			s.Info("LoadBalancer ID is set, fetching loadbalancer details", "loadbalancerid", *loadBalancerID)
			loadBalancer, _, err := s.IBMVPCClient.GetLoadBalancer(&vpcv1.GetLoadBalancerOptions{
				ID: loadBalancerID,
			})
			if err != nil {
				return err
			}
			if infrav1beta2.VPCLoadBalancerState(*loadBalancer.ProvisioningStatus) != infrav1beta2.VPCLoadBalancerStateActive {
				return fmt.Errorf("loadbalancer is not in active state, current state %s", *loadBalancer.ProvisioningStatus)
			}
			state := infrav1beta2.VPCLoadBalancerState(*loadBalancer.ProvisioningStatus)
			loadBalancerStatus := infrav1beta2.VPCLoadBalancerStatus{
				ID:       loadBalancer.ID,
				State:    state,
				Hostname: loadBalancer.Hostname,
			}
			s.SetLoadBalancerStatus(*loadBalancer.Name, loadBalancerStatus)
			continue
		}
		// check VPC subnet exist in cloud
		loadBalancerStatus, err := s.checkLoadBalancer(loadBalancer)
		if err != nil {
			return err
		}
		if loadBalancerStatus != nil {
			s.SetLoadBalancerStatus(loadBalancer.Name, *loadBalancerStatus)
			continue
		}
		// create loadBalancer
		loadBalancerStatus, err = s.createLoadBalancer(loadBalancer)
		if err != nil {
			return err
		}
		s.SetLoadBalancerStatus(loadBalancer.Name, *loadBalancerStatus)
		return nil
	}
	return nil
}

// checkLoadBalancer checks loadBalancer in cloud.
func (s *PowerVSClusterScope) checkLoadBalancer(lb infrav1beta2.VPCLoadBalancerSpec) (*infrav1beta2.VPCLoadBalancerStatus, error) {
	loadBalancer, err := s.IBMVPCClient.GetLoadBalancerByName(lb.Name)
	if err != nil {
		return nil, err
	}
	if loadBalancer == nil {
		return nil, nil
	}
	state := infrav1beta2.VPCLoadBalancerState(*loadBalancer.ProvisioningStatus)
	return &infrav1beta2.VPCLoadBalancerStatus{
		ID:       loadBalancer.ID,
		State:    state,
		Hostname: loadBalancer.Hostname,
	}, nil
}

// createLoadBalancer creates loadBalancer.
func (s *PowerVSClusterScope) createLoadBalancer(lb infrav1beta2.VPCLoadBalancerSpec) (*infrav1beta2.VPCLoadBalancerStatus, error) {
	options := &vpcv1.CreateLoadBalancerOptions{}
	// TODO(karthik-k-n): consider moving resource group id to clusterscope
	// fetch resource group id
	resourceGroupID, err := s.getResourceGroupID()
	if err != nil {
		return nil, fmt.Errorf("error getting id for resource group %s, %w", *s.ResourceGroup(), err)
	}

	options.SetName(lb.Name)
	options.SetIsPublic(true)
	options.SetResourceGroup(&vpcv1.ResourceGroupIdentity{
		ID: &resourceGroupID,
	})

	subnetIds := s.GetVPCSubnetIDs()
	if subnetIds == nil {
		return nil, fmt.Errorf("error subnet required for load balancer creation")
	}
	for _, subnetID := range subnetIds {
		subnet := &vpcv1.SubnetIdentity{
			ID: subnetID,
		}
		options.Subnets = append(options.Subnets, subnet)
	}
	options.SetPools([]vpcv1.LoadBalancerPoolPrototype{
		{
			Algorithm:     core.StringPtr("round_robin"),
			HealthMonitor: &vpcv1.LoadBalancerPoolHealthMonitorPrototype{Delay: core.Int64Ptr(5), MaxRetries: core.Int64Ptr(2), Timeout: core.Int64Ptr(2), Type: core.StringPtr("tcp")},
			// Note: Appending port number to the name, it will be referenced to set target port while adding new pool member
			Name:     core.StringPtr(fmt.Sprintf("%s-pool-%d", lb.Name, s.APIServerPort())),
			Protocol: core.StringPtr("tcp"),
		},
	})

	options.SetListeners([]vpcv1.LoadBalancerListenerPrototypeLoadBalancerContext{
		{
			Protocol: core.StringPtr("tcp"),
			Port:     core.Int64Ptr(int64(s.APIServerPort())),
			DefaultPool: &vpcv1.LoadBalancerPoolIdentityByName{
				Name: core.StringPtr(fmt.Sprintf("%s-pool-%d", lb.Name, s.APIServerPort())),
			},
		},
	})

	for _, additionalListeners := range lb.AdditionalListeners {
		pool := vpcv1.LoadBalancerPoolPrototype{
			Algorithm:     core.StringPtr("round_robin"),
			HealthMonitor: &vpcv1.LoadBalancerPoolHealthMonitorPrototype{Delay: core.Int64Ptr(5), MaxRetries: core.Int64Ptr(2), Timeout: core.Int64Ptr(2), Type: core.StringPtr("tcp")},
			// Note: Appending port number to the name, it will be referenced to set target port while adding new pool member
			Name:     pointer.String(fmt.Sprintf("additional-pool-%d", additionalListeners.Port)),
			Protocol: core.StringPtr("tcp"),
		}
		options.Pools = append(options.Pools, pool)

		listener := vpcv1.LoadBalancerListenerPrototypeLoadBalancerContext{
			Protocol: core.StringPtr("tcp"),
			Port:     core.Int64Ptr(additionalListeners.Port),
			DefaultPool: &vpcv1.LoadBalancerPoolIdentityByName{
				Name: pointer.String(fmt.Sprintf("additional-pool-%d", additionalListeners.Port)),
			},
		}
		options.Listeners = append(options.Listeners, listener)
	}

	loadBalancer, _, err := s.IBMVPCClient.CreateLoadBalancer(options)
	if err != nil {
		return nil, err
	}
	lbState := infrav1beta2.VPCLoadBalancerState(*loadBalancer.ProvisioningStatus)
	return &infrav1beta2.VPCLoadBalancerStatus{
		ID:                loadBalancer.ID,
		State:             lbState,
		Hostname:          loadBalancer.Hostname,
		ControllerCreated: pointer.Bool(true),
	}, nil
}

// COSInstance returns the COS instance reference.
func (s *PowerVSClusterScope) COSInstance() *infrav1beta2.CosInstance {
	return s.IBMPowerVSCluster.Spec.CosInstance
}

// ReconcileCOSInstance reconcile COS bucket.
func (s *PowerVSClusterScope) ReconcileCOSInstance() error {
	if s.COSInstance() == nil || s.COSInstance().Name == "" {
		return nil
	}

	// check COS service instance exist in cloud
	cosServiceInstanceStatus, err := s.checkCOSServiceInstance()
	if err != nil {
		s.Error(err, "error checking cos service instance")
		return err
	}
	if cosServiceInstanceStatus != nil {
		s.SetStatus(COSInstance, infrav1beta2.ResourceReference{ID: cosServiceInstanceStatus.GUID, ControllerCreated: pointer.Bool(false)})
	} else {
		// create COS service instance
		cosServiceInstanceStatus, err = s.createCOSServiceInstance()
		if err != nil {
			s.Error(err, "error creating cos service instance")
			return err
		}
		s.SetStatus(COSInstance, infrav1beta2.ResourceReference{ID: cosServiceInstanceStatus.GUID, ControllerCreated: pointer.Bool(true)})
	}

	props, err := authenticator.GetProperties()
	if err != nil {
		s.Error(err, "error while fetching service properties")
		return err
	}

	apiKey := props["APIKEY"]
	if apiKey == "" {
		fmt.Printf("ibmcloud api key is not provided, set %s environmental variable", "IBMCLOUD_API_KEY")
	}

	cosClient, err := cos.NewService(cos.ServiceOptions{}, s.IBMPowerVSCluster.Spec.CosInstance.BucketRegion, apiKey, *cosServiceInstanceStatus.GUID)
	if err != nil {
		s.Error(err, "error creating cosClient")
		return fmt.Errorf("failed to create cos client: %w", err)
	}
	s.COSClient = cosClient

	// check bucket exist in service instance
	if exist, err := s.checkCOSBucket(); exist {
		return nil
	} else if err != nil {
		s.Error(err, "error checking cos bucket")
		return err
	}

	// create bucket in service instance
	if err := s.createCOSBucket(); err != nil {
		s.Error(err, "error creating cos bucket")
		return err
	}
	return nil
}

func (s *PowerVSClusterScope) checkCOSBucket() (bool, error) {
	if _, err := s.COSClient.GetBucketByName(s.COSInstance().BucketName); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "Forbidden", "NotFound":
				// If the bucket doesn't exist that's ok, we'll try to create it
				return false, nil
			default:
				return false, err
			}
		} else {
			return false, err
		}
	}
	return true, nil
}

func (s *PowerVSClusterScope) createCOSBucket() error {
	input := &s3.CreateBucketInput{
		Bucket: pointer.String(s.COSInstance().BucketName),
	}
	_, err := s.COSClient.CreateBucket(input)
	if err == nil {
		return nil
	}

	aerr, ok := err.(awserr.Error)
	if !ok {
		return fmt.Errorf("error creating COS bucket %w", err)
	}

	switch aerr.Code() {
	// If bucket already exists, all good.
	case s3.ErrCodeBucketAlreadyOwnedByYou:
		return nil
	case s3.ErrCodeBucketAlreadyExists:
		return nil
	default:
		return fmt.Errorf("error creating COS bucket %w", err)
	}
}

func (s *PowerVSClusterScope) checkCOSServiceInstance() (*resourcecontrollerv2.ResourceInstance, error) {
	// check cos service instance
	serviceInstance, err := s.ResourceClient.GetInstanceByName(s.COSInstance().Name, cosResourceID, cosResourcePlanID)
	if err != nil {
		return nil, err
	}
	if serviceInstance == nil {
		s.Info("cos service instance is nil", "name", s.COSInstance().Name)
		return nil, nil
	}
	if *serviceInstance.State != string(infrav1beta2.ServiceInstanceStateActive) {
		s.Info("cos service instance not in active state", "current state", *serviceInstance.State)
		return nil, fmt.Errorf("cos instance not in active state, current state: %s", *serviceInstance.State)
	}
	return serviceInstance, nil
}

func (s *PowerVSClusterScope) createCOSServiceInstance() (*resourcecontrollerv2.ResourceInstance, error) {
	resourceGroupID, err := s.getResourceGroupID()
	if err != nil {
		return nil, fmt.Errorf("error getting id for resource group %s, %w", *s.ResourceGroup(), err)
	}

	// TODO(karthik-k-n)(Doubt): should this be fetched or hardcoded
	// _, servicePlanID, err := s.CatalogClient.GetServiceInfo(powerVSService, powerVSServicePlan)
	// if err != nil {
	//	return nil, fmt.Errorf("error retrieving id info for powervs service %w", err)
	//}

	target := "Global"
	// create service instance
	serviceInstance, _, err := s.ResourceClient.CreateResourceInstance(&resourcecontrollerv2.CreateResourceInstanceOptions{
		Name:           s.GetServiceName(COSInstance),
		Target:         &target,
		ResourceGroup:  &resourceGroupID,
		ResourcePlanID: pointer.String(cosResourcePlanID),
	})
	if err != nil {
		return nil, err
	}
	return serviceInstance, nil
}

// getResourceGroupID retrieving id of resource group.
func (s *PowerVSClusterScope) getResourceGroupID() (string, error) {
	rmv2, err := resourcemanagerv2.NewResourceManagerV2(&resourcemanagerv2.ResourceManagerV2Options{
		Authenticator: s.session.Options.Authenticator,
	})
	if err != nil {
		return "", err
	}
	if rmv2 == nil {
		return "", fmt.Errorf("unable to get resource controller")
	}
	resourceGroup := s.ResourceGroup()
	rmv2ListResourceGroupOpt := resourcemanagerv2.ListResourceGroupsOptions{Name: resourceGroup, AccountID: &s.session.Options.UserAccount}
	resourceGroupListResult, _, err := rmv2.ListResourceGroups(&rmv2ListResourceGroupOpt)
	if err != nil {
		return "", err
	}

	if resourceGroupListResult != nil && len(resourceGroupListResult.Resources) > 0 {
		rg := resourceGroupListResult.Resources[0]
		resourceGroupID := *rg.ID
		return resourceGroupID, nil
	}

	err = fmt.Errorf("could not retrieve resource group id for %s", *resourceGroup)
	return "", err
}

// getVPCRegion returns region associated with VPC zone.
func (s *PowerVSClusterScope) getVPCRegion() *string {
	if s.IBMPowerVSCluster.Spec.VPC != nil {
		return s.IBMPowerVSCluster.Spec.VPC.Region
	}
	// if vpc region is not set try to fetch corresponding region from power vs zone
	if s.Zone() == nil {
		return nil
	}
	region := endpoints.ConstructRegionFromZone(*s.Zone())
	vpcRegion, err := genUtil.VPCRegionForPowerVSRegion(region)
	if err != nil {
		s.Error(err, fmt.Sprintf("failed to fetch vpc region associated with powervs region %s", region))
		return nil
	}
	return &vpcRegion
}

// fetchVPCCRN returns VPC CRN.
func (s *PowerVSClusterScope) fetchVPCCRN() (*string, error) {
	vpcDetails, _, err := s.IBMVPCClient.GetVPC(&vpcv1.GetVPCOptions{
		ID: s.GetVPCID(),
	})
	if err != nil {
		return nil, err
	}
	return vpcDetails.CRN, nil
}

// fetchPowerVSServiceInstanceCRN returns Power VS service instance CRN.
func (s *PowerVSClusterScope) fetchPowerVSServiceInstanceCRN() (*string, error) {
	serviceInstanceID := s.GetServiceInstanceID()
	pvsDetails, _, err := s.ResourceClient.GetResourceInstance(&resourcecontrollerv2.GetResourceInstanceOptions{
		ID: &serviceInstanceID,
	})
	if err != nil {
		return nil, err
	}
	return pvsDetails.CRN, nil
}

// TODO(karthik-k-n): Decide on proper naming format for services.

// GetServiceName returns name of given service type from spec or generate a name for it.
func (s *PowerVSClusterScope) GetServiceName(resourceType ResourceType) *string {
	switch resourceType {
	case ServiceInstance:
		if s.ServiceInstance() == nil || s.ServiceInstance().Name == nil {
			return pointer.String(fmt.Sprintf("%s-serviceInstance", s.InfraCluster().GetName()))
		}
		return s.ServiceInstance().Name
	case Network:
		if s.Network().Name == nil {
			return pointer.String(fmt.Sprintf("DHCPSERVER%s-dhcp_Private", s.InfraCluster().GetName()))
		}
		return s.Network().Name
	case VPC:
		if s.VPC() == nil || s.VPC().Name == nil {
			return pointer.String(fmt.Sprintf("%s-vpc", s.InfraCluster().GetName()))
		}
		return s.VPC().Name
	case TransitGateway:
		if s.TransitGateway() == nil || s.TransitGateway().Name == nil {
			return pointer.String(fmt.Sprintf("%s-transitgateway", s.InfraCluster().GetName()))
		}
		return s.TransitGateway().Name
	case DHCPServer:
		return pointer.String(fmt.Sprintf("%s-dhcp", s.InfraCluster().GetName()))
	case COSInstance:
		if s.COSInstance() == nil || s.COSInstance().Name == "" {
			return pointer.String(fmt.Sprintf("%s-cosinstance", s.InfraCluster().GetName()))
		}
		return &s.COSInstance().Name
	}
	return nil
}

// DeleteLoadBalancer deletes loadBalancer.
func (s *PowerVSClusterScope) DeleteLoadBalancer() error {
	if !s.deleteResource(LoadBalancer) {
		return nil
	}

	for _, lb := range s.IBMPowerVSCluster.Status.LoadBalancers {
		if lb.ID != nil {
			lb, _, err := s.IBMVPCClient.GetLoadBalancer(&vpcv1.GetLoadBalancerOptions{
				ID: lb.ID,
			})

			if err != nil {
				if strings.Contains(err.Error(), "cannot be found") {
					return nil
				}
				return fmt.Errorf("error fetching the load balancer: %w", err)
			}

			if lb != nil && lb.ProvisioningStatus != nil && *lb.ProvisioningStatus != string(infrav1beta2.VPCLoadBalancerStateDeletePending) {
				_, err = s.IBMVPCClient.DeleteLoadBalancer(&vpcv1.DeleteLoadBalancerOptions{
					ID: lb.ID,
				})
				if err != nil {
					s.Error(err, "error deleting the load balancer")
					return err
				}
				s.Info("Load balancer successfully deleted")
			}
		}
	}
	return nil
}

// DeleteVPCSubnet deletes VPC subnet.
func (s *PowerVSClusterScope) DeleteVPCSubnet() error {
	if !s.deleteResource(Subnet) {
		return nil
	}

	for _, subnet := range s.IBMPowerVSCluster.Status.VPCSubnet {
		if subnet.ID == nil {
			continue
		}

		net, _, err := s.IBMVPCClient.GetSubnet(&vpcv1.GetSubnetOptions{
			ID: subnet.ID,
		})

		if err != nil {
			if strings.Contains(err.Error(), "Subnet not found") {
				return nil
			}
			return fmt.Errorf("error fetching the subnet: %w", err)
		}

		_, err = s.IBMVPCClient.DeleteSubnet(&vpcv1.DeleteSubnetOptions{
			ID: net.ID,
		})
		if err != nil {
			return fmt.Errorf("error deleting VPC subnet: %w", err)
		}
		s.Info("VPC subnet successfully deleted")
	}
	return nil
}

// DeleteVPC deletes VPC.
func (s *PowerVSClusterScope) DeleteVPC() error {
	if !s.deleteResource(VPC) {
		return nil
	}

	if s.IBMPowerVSCluster.Status.VPC.ID != nil {
		vpc, _, err := s.IBMVPCClient.GetVPC(&vpcv1.GetVPCOptions{
			ID: s.IBMPowerVSCluster.Status.VPC.ID,
		})

		if err != nil {
			if strings.Contains(err.Error(), "VPC not found") {
				return nil
			}
			return fmt.Errorf("error fetching the VPC: %w", err)
		}

		_, err = s.IBMVPCClient.DeleteVPC(&vpcv1.DeleteVPCOptions{
			ID: vpc.ID,
		})
		if err != nil {
			return fmt.Errorf("error deleting VPC: %w", err)
		}
		s.Info("VPC successfully deleted")
	}
	return nil
}

// DeleteTransitGateway deletes transit gateway.
func (s *PowerVSClusterScope) DeleteTransitGateway() error {
	if !s.deleteResource(TransitGateway) {
		return nil
	}

	if s.IBMPowerVSCluster.Status.TransitGateway.ID != nil {
		tg, _, err := s.TransitGatewayClient.GetTransitGateway(&tgapiv1.GetTransitGatewayOptions{
			ID: s.IBMPowerVSCluster.Status.TransitGateway.ID,
		})

		if err != nil {
			if strings.Contains(err.Error(), "gateway was not found") {
				return nil
			}
			return fmt.Errorf("error fetching the transit gateway: %w", err)
		}

		tgConnections, _, err := s.TransitGatewayClient.ListTransitGatewayConnections(&tgapiv1.ListTransitGatewayConnectionsOptions{
			TransitGatewayID: tg.ID,
		})
		if err != nil {
			return fmt.Errorf("error listing transit gateway connections: %w", err)
		}

		if tgConnections.Connections != nil && len(tgConnections.Connections) > 0 {
			for _, conn := range tgConnections.Connections {
				if conn.Status != nil && *conn.Status != string(infrav1beta2.TransitGatewayStateDeletePending) {
					_, err := s.TransitGatewayClient.DeleteTransitGatewayConnection(&tgapiv1.DeleteTransitGatewayConnectionOptions{
						ID:               conn.ID,
						TransitGatewayID: tg.ID,
					})
					if err != nil {
						return fmt.Errorf("error deleting transit gateway connection: %w", err)
					}
				}
			}
		}

		_, err = s.TransitGatewayClient.DeleteTransitGateway(&tgapiv1.DeleteTransitGatewayOptions{
			ID: s.IBMPowerVSCluster.Status.TransitGateway.ID,
		})

		if err != nil {
			return fmt.Errorf("error deleting transit gateway: %w", err)
		}
		s.Info("Transit gateway successfully deleted")
	}
	return nil
}

// DeleteDHCPServer deletes DHCP server.
func (s *PowerVSClusterScope) DeleteDHCPServer() error {
	if !s.deleteResource(DHCPServer) {
		return nil
	}

	if s.IBMPowerVSCluster.Status.DHCPServer.ID != nil {
		server, err := s.IBMPowerVSClient.GetDHCPServer(*s.IBMPowerVSCluster.Status.DHCPServer.ID)
		if err != nil {
			if strings.Contains(err.Error(), "dhcp server does not exist") {
				return nil
			}
			return fmt.Errorf("error fetching DHCP server: %w", err)
		}

		err = s.IBMPowerVSClient.DeleteDHCPServer(*server.ID)
		if err != nil {
			return fmt.Errorf("error deleting the DHCP server: %w", err)
		}
		s.Info("DHCP server successfully deleted")
	}
	return nil
}

// DeleteServiceInstance deletes service instance.
func (s *PowerVSClusterScope) DeleteServiceInstance() error {
	if !s.deleteResource(ServiceInstance) {
		return nil
	}

	if s.IBMPowerVSCluster.Status.ServiceInstance.ID != nil {
		serviceInstance, _, err := s.ResourceClient.GetResourceInstance(&resourcecontrollerv2.GetResourceInstanceOptions{
			ID: s.IBMPowerVSCluster.Status.ServiceInstance.ID,
		})
		if err != nil {
			return fmt.Errorf("error fetching service instance: %w", err)
		}

		if serviceInstance != nil && *serviceInstance.State == string(infrav1beta2.ServiceInstanceStateRemoved) {
			return nil
		}

		servers, err := s.IBMPowerVSClient.GetAllDHCPServers()
		if err != nil {
			return fmt.Errorf("error fetching networks in the service instance: %w", err)
		}

		if len(servers) > 0 {
			return fmt.Errorf("cannot delete service instance as network is not yet deleted")
		}

		_, err = s.ResourceClient.DeleteResourceInstance(&resourcecontrollerv2.DeleteResourceInstanceOptions{
			ID: serviceInstance.ID,
			//Recursive: pointer.Bool(true),
		})

		if err != nil {
			s.Error(err, "error deleting Power VS service instance")
			return err
		}
		s.Info("Service instance successfully deleted")
	}
	return nil
}

// DeleteCosInstance deletes COS instance.
func (s *PowerVSClusterScope) DeleteCosInstance() error {
	if !s.deleteResource(COSInstance) {
		return nil
	}

	if s.IBMPowerVSCluster.Status.COSInstance.ID != nil {
		cosInstance, _, err := s.ResourceClient.GetResourceInstance(&resourcecontrollerv2.GetResourceInstanceOptions{
			ID: s.IBMPowerVSCluster.Status.COSInstance.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "COS instance unavailable") {
				return nil
			}
			return fmt.Errorf("error fetching COS instance: %w", err)
		}

		if cosInstance != nil && (*cosInstance.State == "pending_reclamation" || *cosInstance.State == string(infrav1beta2.ServiceInstanceStateRemoved)) {
			return nil
		}

		_, err = s.ResourceClient.DeleteResourceInstance(&resourcecontrollerv2.DeleteResourceInstanceOptions{
			ID:        cosInstance.ID,
			Recursive: pointer.Bool(true),
		})
		if err != nil {
			s.Error(err, "error deleting COS service instance")
			return err
		}
		s.Info("COS instance successfully deleted")
	}
	return nil
}

// deleteResource returns true or false to decide on deleting provided resource.
func (s *PowerVSClusterScope) deleteResource(resourceType ResourceType) bool { //nolint:gocyclo
	switch resourceType {
	case LoadBalancer:
		lbs := s.IBMPowerVSCluster.Status.LoadBalancers
		if lbs == nil {
			return false
		}
		for _, lb := range lbs {
			if lb.ControllerCreated == nil || !*lb.ControllerCreated {
				return false
			}
		}
		return true
	case Subnet:
		subnets := s.IBMPowerVSCluster.Status.VPCSubnet
		if subnets == nil {
			return false
		}
		for _, net := range subnets {
			if net.ControllerCreated == nil || !*net.ControllerCreated {
				return false
			}
		}
		return true
	case VPC:
		vpcStatus := s.IBMPowerVSCluster.Status.VPC
		if vpcStatus == nil || vpcStatus.ControllerCreated == nil || !*vpcStatus.ControllerCreated {
			return false
		}
		return true
	case ServiceInstance:
		serviceInstance := s.IBMPowerVSCluster.Status.ServiceInstance
		if serviceInstance == nil || serviceInstance.ControllerCreated == nil || !*serviceInstance.ControllerCreated {
			return false
		}
		return true
	case TransitGateway:
		transitGateway := s.IBMPowerVSCluster.Status.TransitGateway
		if transitGateway == nil || transitGateway.ControllerCreated == nil || !*transitGateway.ControllerCreated {
			return false
		}
		return true
	case DHCPServer:
		dhcpServer := s.IBMPowerVSCluster.Status.DHCPServer
		if dhcpServer == nil || dhcpServer.ControllerCreated == nil || !*dhcpServer.ControllerCreated {
			return false
		}
		return true
	case COSInstance:
		cosInstance := s.IBMPowerVSCluster.Status.COSInstance
		if cosInstance == nil || cosInstance.ControllerCreated == nil || !*cosInstance.ControllerCreated {
			return false
		}
		return true
	}
	return false
}
