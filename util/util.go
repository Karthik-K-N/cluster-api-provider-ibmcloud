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

package util

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1beta2 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
)

const (
	//PowerVSZone = "dal10"
	//VPCZone     = "us-south-1"

	PowerVSZone = "dal10"
	VPCZone     = "us-south-1"
	VPCRegion   = "us-south"
)

// GetClusterByName finds and return a Cluster object using the specified params.
func GetClusterByName(ctx context.Context, c client.Client, namespace, name string) (*infrav1beta2.IBMPowerVSCluster, error) {
	cluster := &infrav1beta2.IBMPowerVSCluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.Get(ctx, key, cluster); err != nil {
		return nil, fmt.Errorf("failed to get Cluster/%s: %w", name, err)
	}

	return cluster, nil
}

// ConstructVPCRegionFromZone returns region based on location/zone.
func ConstructVPCRegionFromZone(zone string) string {
	var regex string
	if strings.Contains(zone, "-") {
		// it's a region or AZ
		regex = "-[0-9]+$"
	} else {
		// it's a datacenter
		regex = "[0-9]+$"
	}

	reg, _ := regexp.Compile(regex)
	return reg.ReplaceAllString(zone, "")
}

// Region describes respective IBM Cloud COS region, VPC region and Zones associated with a region in Power VS.
type Region struct {
	Description string
	VPCRegion   string
	COSRegion   string
	Zones       []string
	VPCZones    []string
	SysTypes    []string
}

// Regions provides a mapping between Power VS and IBM Cloud VPC and IBM COS regions.
var Regions = map[string]Region{
	"dal": {
		Description: "Dallas, USA",
		VPCRegion:   "us-south",
		COSRegion:   "us-south",
		Zones: []string{
			"dal10",
			"dal12",
		},
		SysTypes: []string{"s922", "e980"},
		VPCZones: []string{"us-south-1", "us-south-2", "us-south-3"},
	},
	"eu-de": {
		Description: "Frankfurt, Germany",
		VPCRegion:   "eu-de",
		COSRegion:   "eu-de",
		Zones: []string{
			"eu-de-1",
			"eu-de-2",
		},
		SysTypes: []string{"s922", "e980"},
		VPCZones: []string{"eu-de-2", "eu-de-3"},
	},
	"lon": {
		Description: "London, UK.",
		VPCRegion:   "eu-gb",
		COSRegion:   "eu-gb",
		Zones: []string{
			"lon04",
			"lon06",
		},
		SysTypes: []string{"s922", "e980"},
		VPCZones: []string{"eu-gb-1", "eu-gb-3"},
	},
	"mad": {
		Description: "Madrid, Spain",
		VPCRegion:   "eu-es",
		COSRegion:   "eu-de", // @HACK - PowerVS says COS not supported in this region
		Zones: []string{
			"mad02",
			"mad04",
		},
		SysTypes: []string{"s1022"},
		VPCZones: []string{"eu-es-1", "eu-es-2"},
	},
	"mon": {
		Description: "Montreal, Canada",
		VPCRegion:   "ca-tor",
		COSRegion:   "ca-tor",
		Zones:       []string{"mon01"},
		SysTypes:    []string{"s922", "e980"},
	},
	"osa": {
		Description: "Osaka, Japan",
		VPCRegion:   "jp-osa",
		COSRegion:   "jp-osa",
		Zones:       []string{"osa21"},
		SysTypes:    []string{"s922", "e980"},
		VPCZones:    []string{"jp-osa-1"},
	},
	"syd": {
		Description: "Sydney, Australia",
		VPCRegion:   "au-syd",
		COSRegion:   "au-syd",
		Zones: []string{
			"syd04",
			"syd05",
		},
		SysTypes: []string{"s922", "e980"},
		VPCZones: []string{"au-syd-2", "au-syd-3"},
	},
	"sao": {
		Description: "São Paulo, Brazil",
		VPCRegion:   "br-sao",
		COSRegion:   "br-sao",
		Zones: []string{
			"sao01",
			"sao04",
		},
		SysTypes: []string{"s922", "e980"},
		VPCZones: []string{"br-sao-1", "br-sao-2"},
	},
	"tok": {
		Description: "Tokyo, Japan",
		VPCRegion:   "jp-tok",
		COSRegion:   "jp-tok",
		Zones:       []string{"tok04"},
		SysTypes:    []string{"s922", "e980"},
		VPCZones:    []string{"jp-tok-2"},
	},
	"us-east": {
		Description: "Washington DC, USA",
		VPCRegion:   "us-east",
		COSRegion:   "us-east",
		Zones:       []string{"us-east"},
		SysTypes:    []string{}, // Missing
		VPCZones:    []string{"us-east-1", "us-east-2", "us-east-3"},
	},
	"wdc": {
		Description: "Washington DC, USA",
		VPCRegion:   "us-east",
		COSRegion:   "us-east",
		Zones: []string{
			"wdc06",
			"wdc07",
		},
		SysTypes: []string{"s922", "e980"},
		VPCZones: []string{"us-east-1", "us-east-2", "us-east-3"},
	},
}

// VPCRegionForPowerVSRegion returns the VPC region for the specified PowerVS region.
func VPCRegionForPowerVSRegion(region string) (string, error) {
	if r, ok := Regions[region]; ok {
		return r.VPCRegion, nil
	}
	return "", fmt.Errorf("VPC region corresponding to a PowerVS region %s not found ", region)
}

// VPCZonesForPowerVSRegion returns the VPC zones associated with Power VS region
func VPCZonesForPowerVSRegion(region string) ([]string, error) {
	if r, ok := Regions[region]; ok {
		return r.VPCZones, nil
	}
	return nil, fmt.Errorf("VPC zones corresponding to a PowerVS region %s not found ", region)
}
