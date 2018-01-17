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

package kops

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is a specific cluster wrapper
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a list of clusters
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Cluster `json:"items"`
}

// ClusterSpec defines the configuration for a cluster
type ClusterSpec struct {
	// The Channel we are following
	Channel string `json:"channel,omitempty"`
	// ConfigBase is the path where we store configuration for the cluster
	// This might be different that the location when the cluster spec itself is stored,
	// both because this must be accessible to the cluster,
	// and because it might be on a different cloud or storage system (etcd vs S3)
	ConfigBase string `json:"configBase,omitempty"`
	// The CloudProvider to use (aws or gce)
	CloudProvider string `json:"cloudProvider,omitempty"`
	// The version of kubernetes to install (optional, and can be a "spec" like stable)
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// Configuration of subnets we are targeting
	Subnets []ClusterSubnetSpec `json:"subnets,omitempty"`
	// Project is the cloud project we should use, required on GCE
	Project string `json:"project,omitempty"`
	// MasterPublicName is the external DNS name for the master nodes
	MasterPublicName string `json:"masterPublicName,omitempty"`
	// MasterInternalName is the internal DNS name for the master nodes
	MasterInternalName string `json:"masterInternalName,omitempty"`
	// NetworkCIDR is the CIDR used for the AWS VPC / GCE Network, or otherwise allocated to k8s
	// This is a real CIDR, not the internal k8s network
	// On AWS, it maps to the VPC CIDR.  It is not required on GCE.
	NetworkCIDR string `json:"networkCIDR,omitempty"`
	// AdditionalNetworkCIDRs is a list of aditional CIDR used for the AWS VPC
	// or otherwise allocated to k8s. This is a real CIDR, not the internal k8s network
	// On AWS, it maps to any aditional CIDRs added to a VPC.
	AdditionalNetworkCIDRs []string `json:"additionalNetworkCIDRs,omitempty"`
	// NetworkID is an identifier of a network, if we want to reuse/share an existing network (e.g. an AWS VPC)
	NetworkID             string `json:"networkID,omitempty"`
	EnableInternetGateway bool   `json:"enableInternetGateway,omitempty"`
	SkipUpdateSubnetName  bool   `json:"skipUpdateSubnetName,omitempty"`
	// Topology defines the type of network topology to use on the cluster - default public
	// This is heavily weighted towards AWS for the time being, but should also be agnostic enough
	// to port out to GCE later if needed
	Topology *TopologySpec `json:"topology,omitempty"`
	// SecretStore is the VFS path to where secrets are stored
	SecretStore string `json:"secretStore,omitempty"`
	// KeyStore is the VFS path to where SSL keys and certificates are stored
	KeyStore string `json:"keyStore,omitempty"`
	// ConfigStore is the VFS path to where the configuration (Cluster, InstanceGroups etc) is stored
	ConfigStore string `json:"configStore,omitempty"`
	// DNSZone is the DNS zone we should use when configuring DNS
	// This is because some clouds let us define a managed zone foo.bar, and then have
	// kubernetes.dev.foo.bar, without needing to define dev.foo.bar as a hosted zone.
	// DNSZone will probably be a suffix of the MasterPublicName and MasterInternalName
	// Note that DNSZone can either by the host name of the zone (containing dots),
	// or can be an identifier for the zone.
	DNSZone string `json:"dnsZone,omitempty"`
	// AdditionalSANs adds additional Subject Alternate Names to apiserver cert that kops generates
	AdditionalSANs []string `json:"additionalSans,omitempty"`
	// ClusterDNSDomain is the suffix we use for internal DNS names (normally cluster.local)
	ClusterDNSDomain string `json:"clusterDNSDomain,omitempty"`
	// ServiceClusterIPRange is the CIDR, from the internal network, where we allocate IPs for services
	ServiceClusterIPRange string `json:"serviceClusterIPRange,omitempty"`
	// NonMasqueradeCIDR is the CIDR for the internal k8s network (on which pods & services live)
	// It cannot overlap ServiceClusterIPRange
	NonMasqueradeCIDR string `json:"nonMasqueradeCIDR,omitempty"`
	// SSHAccess is a list of the CIDRs that can access SSH.
	SSHAccess []string `json:"sshAccess,omitempty"`
	// NodePortAccess is a list of the CIDRs that can access the node ports range (30000-32767).
	NodePortAccess []string `json:"nodePortAccess,omitempty"`
	// HTTPProxy defines connection information to support use of a private cluster behind an forward HTTP Proxy
	EgressProxy *EgressProxySpec `json:"egressProxy,omitempty"`
	// SSHKeyName specifies a preexisting SSH key to use
	SSHKeyName string `json:"sshKeyName,omitempty"`
	// KubernetesAPIAccess is a list of the CIDRs that can access the Kubernetes API endpoint (master HTTPS)
	KubernetesAPIAccess []string `json:"kubernetesApiAccess,omitempty"`
	// IsolatesMasters determines whether we should lock down masters so that they are not on the pod network.
	// true is the kube-up behaviour, but it is very surprising: it means that daemonsets only work on the master
	// if they have hostNetwork=true.
	// false is now the default, and it will:
	//  * give the master a normal PodCIDR
	//  * run kube-proxy on the master
	//  * enable debugging handlers on the master, so kubectl logs works
	IsolateMasters *bool `json:"isolateMasters,omitempty"`
	// UpdatePolicy determines the policy for applying upgrades automatically.
	// Valid values:
	//   'external' do not apply updates automatically - they are applied manually or by an external system
	//   missing: default policy (currently OS security upgrades that do not require a reboot)
	UpdatePolicy *string `json:"updatePolicy,omitempty"`
	// Additional policies to add for roles
	AdditionalPolicies *map[string]string `json:"additionalPolicies,omitempty"`
	// A collection of files assets for deployed cluster wide
	FileAssets []FileAssetSpec `json:"fileAssets,omitempty"`
	// EtcdClusters stores the configuration for each cluster
	EtcdClusters []*EtcdClusterSpec `json:"etcdClusters,omitempty"`
	// Component configurations
	Docker                         *DockerConfig                 `json:"docker,omitempty"`
	KubeDNS                        *KubeDNSConfig                `json:"kubeDNS,omitempty"`
	KubeAPIServer                  *KubeAPIServerConfig          `json:"kubeAPIServer,omitempty"`
	KubeControllerManager          *KubeControllerManagerConfig  `json:"kubeControllerManager,omitempty"`
	ExternalCloudControllerManager *CloudControllerManagerConfig `json:"cloudControllerManager,omitempty"`
	KubeScheduler                  *KubeSchedulerConfig          `json:"kubeScheduler,omitempty"`
	KubeProxy                      *KubeProxyConfig              `json:"kubeProxy,omitempty"`
	Kubelet                        *KubeletConfigSpec            `json:"kubelet,omitempty"`
	MasterKubelet                  *KubeletConfigSpec            `json:"masterKubelet,omitempty"`
	CloudConfig                    *CloudConfiguration           `json:"cloudConfig,omitempty"`
	ExternalDNS                    *ExternalDNSConfig            `json:"externalDns,omitempty"`

	// Networking configuration
	Networking *NetworkingSpec `json:"networking,omitempty"`
	// API field controls how the API is exposed outside the cluster
	API *AccessSpec `json:"api,omitempty"`
	// Authentication field controls how the cluster is configured for authentication
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`
	// Authorization field controls how the cluster is configured for authorization
	Authorization *AuthorizationSpec `json:"authorization,omitempty"`
	// Tags for AWS instance groups
	CloudLabels map[string]string `json:"cloudLabels,omitempty"`
	// Hooks for custom actions e.g. on first installation
	Hooks []HookSpec `json:"hooks,omitempty"`
	// Assets is alternative locations for files and containers; the API under construction, will remove this comment once this API is fully functional.
	Assets *Assets `json:"assets,omitempty"`
	// IAM field adds control over the IAM security policies applied to resources
	IAM *IAMSpec `json:"iam,omitempty"`
	// EncryptionConfig controls if encryption is enabled
	EncryptionConfig *bool `json:"encryptionConfig,omitempty"`
}

// FileAssetSpec defines the structure for a file asset
type FileAssetSpec struct {
	// Name is a shortened reference to the asset
	Name string `json:"name,omitempty"`
	// Path is the location this file should reside
	Path string `json:"path,omitempty"`
	// Roles is a list of roles the file asset should be applied, defaults to all
	Roles []InstanceGroupRole `json:"roles,omitempty"`
	// Content is the contents of the file
	Content string `json:"content,omitempty"`
	// IsBase64 indicates the contents is base64 encoded
	IsBase64 bool `json:"isBase64,omitempty"`
}

// Assets defined the privately hosted assets
type Assets struct {
	// ContainerRegistry is a url for to a docker registry
	ContainerRegistry *string `json:"containerRegistry,omitempty"`
	// FileRepository is the url for a private file serving repository
	FileRepository *string `json:"fileRepository,omitempty"`
}

// IAMSpec adds control over the IAM security policies applied to resources
type IAMSpec struct {
	Legacy                 bool `json:"legacy"`
	AllowContainerRegistry bool `json:"allowContainerRegistry,omitempty"`
}

// HookSpec is a definition hook
type HookSpec struct {
	// Name is an optional name for the hook, otherwise the name is kops-hook-<index>
	Name string `json:"name,omitempty"`
	// Disabled indicates if you want the unit switched off
	Disabled bool `json:"disabled,omitempty"`
	// Roles is an optional list of roles the hook should be rolled out to, defaults to all
	Roles []InstanceGroupRole `json:"roles,omitempty"`
	// Requires is a series of systemd units the action requires
	Requires []string `json:"requires,omitempty"`
	// Before is a series of systemd units which this hook must run before
	Before []string `json:"before,omitempty"`
	// ExecContainer is the image itself
	ExecContainer *ExecContainerAction `json:"execContainer,omitempty"`
	// Manifest is a raw systemd unit file
	Manifest string `json:"manifest,omitempty"`
}

// ExecContainerAction defines an hood action
type ExecContainerAction struct {
	// Image is the docker image
	Image string `json:"image,omitempty"`
	// Command is the command supplied to the above image
	Command []string `json:"command,omitempty"`
	// Environment is a map of environment variables added to the hook
	Environment map[string]string `json:"environment,omitempty"`
}

type AuthenticationSpec struct {
	Kopeio *KopeioAuthenticationSpec `json:"kopeio,omitempty"`
}

func (s *AuthenticationSpec) IsEmpty() bool {
	return s.Kopeio == nil
}

type KopeioAuthenticationSpec struct {
}

type AuthorizationSpec struct {
	AlwaysAllow *AlwaysAllowAuthorizationSpec `json:"alwaysAllow,omitempty"`
	RBAC        *RBACAuthorizationSpec        `json:"rbac,omitempty"`
}

func (s *AuthorizationSpec) IsEmpty() bool {
	return s.RBAC == nil && s.AlwaysAllow == nil
}

type RBACAuthorizationSpec struct {
}

type AlwaysAllowAuthorizationSpec struct {
}

// AccessSpec provides configuration details related to kubeapi dns and ELB access
type AccessSpec struct {
	// DNS wil be used to provide config on kube-apiserver elb dns
	DNS *DNSAccessSpec `json:"dns,omitempty"`
	// LoadBalancer is the configuration for the kube-apiserver ELB
	LoadBalancer *LoadBalancerAccessSpec `json:"loadBalancer,omitempty"`
}

func (s *AccessSpec) IsEmpty() bool {
	return s.DNS == nil && s.LoadBalancer == nil
}

type DNSAccessSpec struct {
}

// LoadBalancerType string describes LoadBalancer types (public, internal)
type LoadBalancerType string

const (
	LoadBalancerTypePublic   LoadBalancerType = "Public"
	LoadBalancerTypeInternal LoadBalancerType = "Internal"
)

// LoadBalancerAccessSpec provides configuration details related to API LoadBalancer and its access
type LoadBalancerAccessSpec struct {
	Type                     LoadBalancerType `json:"type,omitempty"`
	IdleTimeoutSeconds       *int64           `json:"idleTimeoutSeconds,omitempty"`
	AdditionalSecurityGroups []string         `json:"additionalSecurityGroups,omitempty"`
}

// KubeDNSConfig defines the kube dns configuration
type KubeDNSConfig struct {
	// Image is the name of the docker image to run
	// Deprecated as this is now in the addon
	Image string `json:"image,omitempty"`
	// Replicas is the number of pod replicas
	// Deprecated as this is now in the addon, and controlled by autoscaler
	Replicas int `json:"replicas,omitempty"`
	// Domain is the dns domain
	Domain string `json:"domain,omitempty"`
	// ServerIP is the server ip
	ServerIP string `json:"serverIP,omitempty"`
}

// ExternalDNSConfig are options of the dns-controller
type ExternalDNSConfig struct {
	// Disable indicates we do not wish to run the dns-controller addon
	Disable bool `json:"disable,omitempty"`
	// WatchIngress indicates you want the dns-controller to watch and create dns entries for ingress resources
	WatchIngress *bool `json:"watchIngress,omitempty"`
	// WatchNamespace is namespace to watch, detaults to all (use to control whom can creates dns entries)
	WatchNamespace string `json:"watchNamespace,omitempty"`
}

// EtcdClusterSpec is the etcd cluster specification
type EtcdClusterSpec struct {
	// Name is the name of the etcd cluster (main, events etc)
	Name string `json:"name,omitempty"`
	// Members stores the configurations for each member of the cluster (including the data volume)
	Members []*EtcdMemberSpec `json:"etcdMembers,omitempty"`
	// EnableEtcdTLS indicates the etcd service should use TLS between peers and clients
	EnableEtcdTLS bool `json:"enableEtcdTLS,omitempty"`
	// Version is the version of etcd to run i.e. 2.1.2, 3.0.17 etcd
	Version string `json:"version,omitempty"`
	// LeaderElectionTimeout is the time (in milliseconds) for an etcd leader election timeout
	LeaderElectionTimeout *metav1.Duration `json:"leaderElectionTimeout,omitempty"`
	// HeartbeatInterval is the time (in milliseconds) for an etcd heartbeat interval
	HeartbeatInterval *metav1.Duration `json:"heartbeatInterval,omitempty"`
}

// EtcdMemberSpec is a specification for a etcd member
type EtcdMemberSpec struct {
	// Name is the name of the member within the etcd cluster
	Name string `json:"name,omitempty"`
	// InstanceGroup is the instanceGroup this volume is associated
	InstanceGroup *string `json:"instanceGroup,omitempty"`
	// VolumeType is the underlining cloud storage class
	VolumeType *string `json:"volumeType,omitempty"`
	// VolumeSize is the underlining cloud volume size
	VolumeSize *int32 `json:"volumeSize,omitempty"`
	// KmsKeyId is a AWS KMS ID used to encrypt the volume
	KmsKeyId *string `json:"kmsKeyId,omitempty"`
	// EncryptedVolume indicates you want to encrypt the volume
	EncryptedVolume *bool `json:"encryptedVolume,omitempty"`
}

// SubnetType string describes subnet types (public, private, utility)
type SubnetType string

const (
	// SubnetTypePublic means the subnet is public
	SubnetTypePublic SubnetType = "Public"
	// SubnetTypePrivate means the subnet has no public address or is natted
	SubnetTypePrivate SubnetType = "Private"
	// SubnetTypeUtility mean the subnet is used for utility services, such as the bastion
	SubnetTypeUtility SubnetType = "Utility"
)

// ClusterSubnetSpec defines a subnet
type ClusterSubnetSpec struct {
	// Name is the name of the subnet
	Name string `json:"name,omitempty"`
	// CIDR is the network cidr of the subnet
	CIDR string `json:"cidr,omitempty"`
	// Zone is the zone the subnet is in, set for subnets that are zonally scoped
	Zone string `json:"zone,omitempty"`
	// Region is the region the subnet is in, set for subnets that are regionally scoped
	Region string `json:"region,omitempty"`
	// ProviderID is the cloud provider id for the objects associated with the zone (the subnet on AWS)
	ProviderID string `json:"id,omitempty"`
	// Egress defines the method of traffic egress for this subnet
	Egress string `json:"egress,omitempty"`
	// Type define which one if the internal types (public, utility, private) the network is
	Type SubnetType `json:"type,omitempty"`
}

type EgressProxySpec struct {
	HTTPProxy     HTTPProxy `json:"httpProxy,omitempty"`
	ProxyExcludes string    `json:"excludes,omitempty"`
}

type HTTPProxy struct {
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`
	// TODO #3070
	// User     string `json:"user,omitempty"`
	// Password string `json:"password,omitempty"`
}

// FillDefaults populates default values.
// This is different from PerformAssignments, because these values are changeable, and thus we don't need to
// store them (i.e. we don't need to 'lock them')
func (c *Cluster) FillDefaults() error {
	// Topology support
	if c.Spec.Topology == nil {
		c.Spec.Topology = &TopologySpec{Masters: TopologyPublic, Nodes: TopologyPublic}
		c.Spec.Topology.DNS = &DNSSpec{Type: DNSTypePublic}
	}

	if c.Spec.Networking == nil {
		c.Spec.Networking = &NetworkingSpec{}
	}

	// TODO move this into networking.go :(
	if c.Spec.Networking.Classic != nil {
		// OK
	} else if c.Spec.Networking.Kubenet != nil {
		// OK
	} else if c.Spec.Networking.CNI != nil {
		// OK
	} else if c.Spec.Networking.External != nil {
		// OK
	} else if c.Spec.Networking.Kopeio != nil {
		// OK
	} else if c.Spec.Networking.Weave != nil {
		// OK
	} else if c.Spec.Networking.Flannel != nil {
		// OK
	} else if c.Spec.Networking.Calico != nil {
		// OK
	} else if c.Spec.Networking.Canal != nil {
		// OK
	} else if c.Spec.Networking.Kuberouter != nil {
		// OK
	} else if c.Spec.Networking.Romana != nil {
		// OK
	} else if c.Spec.Networking.AmazonVPC != nil {
		// OK
	} else {
		// No networking model selected; choose Kubenet
		c.Spec.Networking.Kubenet = &KubenetNetworkingSpec{}
	}

	if c.Spec.Channel == "" {
		c.Spec.Channel = DefaultChannel
	}

	if c.ObjectMeta.Name == "" {
		return fmt.Errorf("cluster Name not set in FillDefaults")
	}

	if c.Spec.MasterInternalName == "" {
		c.Spec.MasterInternalName = "api.internal." + c.ObjectMeta.Name
	}

	if c.Spec.MasterPublicName == "" {
		c.Spec.MasterPublicName = "api." + c.ObjectMeta.Name
	}

	return nil
}

// SharedVPC is a simple helper function which makes the templates for a shared VPC clearer
func (c *Cluster) SharedVPC() bool {
	return c.Spec.NetworkID != ""
}
