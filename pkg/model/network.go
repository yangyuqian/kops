/*
Copyright 2016 The Kubernetes Authors.

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

package model

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/awstasks"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/aws"
)

// NetworkModelBuilder configures network objects
type NetworkModelBuilder struct {
	*KopsModelContext
	Lifecycle *fi.Lifecycle
}

var _ fi.ModelBuilder = &NetworkModelBuilder{}

func (b *NetworkModelBuilder) Build(c *fi.ModelBuilderContext) error {
	sharedVPC := b.Cluster.SharedVPC()
	vpcName := b.ClusterName()

	// VPC that holds everything for the cluster
	{
		tags := b.CloudTags(vpcName, sharedVPC)

		t := &awstasks.VPC{
			Name:             s(vpcName),
			Lifecycle:        b.Lifecycle,
			Shared:           fi.Bool(sharedVPC),
			EnableDNSSupport: fi.Bool(true),
			Tags:             tags,
		}

		if sharedVPC && b.IsKubernetesGTE("1.5") {
			// If we're running k8s 1.5, and we have e.g.  --kubelet-preferred-address-types=InternalIP,Hostname,ExternalIP,LegacyHostIP
			// then we don't need EnableDNSHostnames any more
			glog.V(4).Infof("Kubernetes version %q; skipping EnableDNSHostnames requirement on VPC", b.KubernetesVersion())
		} else {
			// In theory we don't need to enable it for >= 1.5,
			// but seems safer to stick with existing behaviour

			t.EnableDNSHostnames = fi.Bool(true)
		}

		if b.Cluster.Spec.NetworkID != "" {
			t.ID = s(b.Cluster.Spec.NetworkID)
		}

		if b.Cluster.Spec.NetworkCIDR != "" {
			t.CIDR = s(b.Cluster.Spec.NetworkCIDR)
		}
		c.AddTask(t)
	}

	if !sharedVPC {
		dhcp := &awstasks.DHCPOptions{
			Name:              s(b.ClusterName()),
			Lifecycle:         b.Lifecycle,
			DomainNameServers: s("AmazonProvidedDNS"),
		}
		if b.Region == "us-east-1" {
			dhcp.DomainName = s("ec2.internal")
		} else {
			dhcp.DomainName = s(b.Region + ".compute.internal")
		}
		c.AddTask(dhcp)

		c.AddTask(&awstasks.VPCDHCPOptionsAssociation{
			Name:        s(b.ClusterName()),
			Lifecycle:   b.Lifecycle,
			VPC:         b.LinkToVPC(),
			DHCPOptions: dhcp,
		})
	} else {
		// TODO: would be good to create these as shared, to verify them
	}

	allSubnetsShared := true
	for i := range b.Cluster.Spec.Subnets {
		subnetSpec := &b.Cluster.Spec.Subnets[i]
		sharedSubnet := subnetSpec.ProviderID != ""
		if !sharedSubnet {
			allSubnetsShared = false
		}
	}

	// We always have a public route table, though for private networks it is only used for NGWs and ELBs
	var publicRouteTable *awstasks.RouteTable
	{
		var igw *awstasks.InternetGateway
		if b.Cluster.Spec.EnableInternetGateway {
			// The internet gateway is the main entry point to the cluster.
			igw = &awstasks.InternetGateway{
				Name:      s(b.ClusterName()),
				Lifecycle: b.Lifecycle,
				VPC:       b.LinkToVPC(),
				Shared:    fi.Bool(sharedVPC),
			}
			c.AddTask(igw)
		}

		if !allSubnetsShared && b.Cluster.Spec.EnableInternetGateway {
			publicRouteTable = &awstasks.RouteTable{
				Name:      s(b.ClusterName()),
				Lifecycle: b.Lifecycle,

				VPC: b.LinkToVPC(),
			}
			c.AddTask(publicRouteTable)

			// TODO: Validate when allSubnetsShared
			c.AddTask(&awstasks.Route{
				Name:            s("0.0.0.0/0"),
				Lifecycle:       b.Lifecycle,
				CIDR:            s("0.0.0.0/0"),
				RouteTable:      publicRouteTable,
				InternetGateway: igw,
			})
		}
	}

	privateZones := sets.NewString()

	for i := range b.Cluster.Spec.Subnets {
		subnetSpec := &b.Cluster.Spec.Subnets[i]
		sharedSubnet := subnetSpec.ProviderID != ""
		subnetName := subnetSpec.Name + "." + b.ClusterName()
		if b.Cluster.Spec.SharedSubnetName {
			subnetName = subnetSpec.Name
		}

		tags := b.CloudTags(subnetName, sharedSubnet)

		// Apply tags so that Kubernetes knows which subnets should be used for internal/external ELBs
		switch subnetSpec.Type {
		case kops.SubnetTypePublic, kops.SubnetTypeUtility:
			tags[aws.TagNameSubnetPublicELB] = "1"

		case kops.SubnetTypePrivate:
			tags[aws.TagNameSubnetInternalELB] = "1"

		default:
			glog.V(2).Infof("unable to properly tag subnet %q because it has unknown type %q. Load balancers may be created in incorrect subnets", subnetSpec.Name, subnetSpec.Type)
		}

		tags["SubnetType"] = string(subnetSpec.Type)

		subnet := &awstasks.Subnet{
			Name:             s(subnetName),
			Lifecycle:        b.Lifecycle,
			VPC:              b.LinkToVPC(),
			AvailabilityZone: s(subnetSpec.Zone),
			CIDR:             s(subnetSpec.CIDR),
			Shared:           fi.Bool(sharedSubnet),
			Tags:             tags,
		}

		if subnetSpec.ProviderID != "" {
			subnet.ID = s(subnetSpec.ProviderID)
		}
		c.AddTask(subnet)

		switch subnetSpec.Type {
		case kops.SubnetTypePublic, kops.SubnetTypeUtility:
			if !sharedSubnet && publicRouteTable != nil {
				c.AddTask(&awstasks.RouteTableAssociation{
					Name:       s(subnetSpec.Name + "." + b.ClusterName()),
					Lifecycle:  b.Lifecycle,
					RouteTable: publicRouteTable,
					Subnet:     subnet,
				})
			}

		case kops.SubnetTypePrivate:
			// Private subnets get a Network Gateway, and their own route table to associate them with the network gateway

			if !sharedSubnet {
				// Private Subnet Route Table Associations
				//
				// Map the Private subnet to the Private route table
				c.AddTask(&awstasks.RouteTableAssociation{
					Name:       s("private-" + subnetSpec.Name + "." + b.ClusterName()),
					Lifecycle:  b.Lifecycle,
					RouteTable: b.LinkToPrivateRouteTableInZone(subnetSpec.Zone),
					Subnet:     subnet,
				})

				// TODO: validate even if shared?
				privateZones.Insert(subnetSpec.Zone)
			}
		default:
			return fmt.Errorf("subnet %q has unknown type %q", subnetSpec.Name, subnetSpec.Type)
		}
	}

	// Loop over zones
	for i, zone := range privateZones.List() {

		utilitySubnet, err := b.LinkToUtilitySubnetInZone(zone)
		if err != nil {
			return err
		}

		var ngw *awstasks.NatGateway
		if b.Cluster.Spec.Subnets[i].Egress != "" {
			if strings.Contains(b.Cluster.Spec.Subnets[i].Egress, "nat-") {

				ngw = &awstasks.NatGateway{
					Name:                 s(zone + "." + b.ClusterName()),
					Lifecycle:            b.Lifecycle,
					Subnet:               utilitySubnet,
					ID:                   s(b.Cluster.Spec.Subnets[i].Egress),
					AssociatedRouteTable: b.LinkToPrivateRouteTableInZone(zone),
					// If we're here, it means this NatGateway was specified, so we are Shared
					Shared: fi.Bool(true),
				}

				c.AddTask(ngw)

			} else {
				return fmt.Errorf("kops currently only supports re-use of NAT Gateways. We will support more eventually! Please see https://github.com/kubernetes/kops/issues/1530")
			}

		} else {

			// Every NGW needs a public (Elastic) IP address, every private
			// subnet needs a NGW, lets create it. We tie it to a subnet
			// so we can track it in AWS
			var eip = &awstasks.ElasticIP{}

			eip = &awstasks.ElasticIP{
				Name:                           s(zone + "." + b.ClusterName()),
				Lifecycle:                      b.Lifecycle,
				AssociatedNatGatewayRouteTable: b.LinkToPrivateRouteTableInZone(zone),
			}

			c.AddTask(eip)
			// NAT Gateway
			//
			// All private subnets will need a NGW, one per zone
			//
			// The instances in the private subnet can access the Internet by
			// using a network address translation (NAT) gateway that resides
			// in the public subnet.

			//var ngw = &awstasks.NatGateway{}
			ngw = &awstasks.NatGateway{
				Name:                 s(zone + "." + b.ClusterName()),
				Lifecycle:            b.Lifecycle,
				Subnet:               utilitySubnet,
				ElasticIP:            eip,
				AssociatedRouteTable: b.LinkToPrivateRouteTableInZone(zone),
			}
			c.AddTask(ngw)
		}

		// Private Route Table
		//
		// The private route table that will route to the NAT Gateway
		rt := &awstasks.RouteTable{
			Name:      s(b.NamePrivateRouteTableInZone(zone)),
			VPC:       b.LinkToVPC(),
			Lifecycle: b.Lifecycle,
		}
		c.AddTask(rt)

		// Private Routes
		//
		// Routes for the private route table.
		// Will route to the NAT Gateway
		c.AddTask(&awstasks.Route{
			Name:       s("private-" + zone + "-0.0.0.0/0"),
			Lifecycle:  b.Lifecycle,
			CIDR:       s("0.0.0.0/0"),
			RouteTable: rt,
			NatGateway: ngw,
		})

	}

	return nil
}
