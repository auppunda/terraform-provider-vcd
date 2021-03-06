package vcd

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_USER", nil),
				Description: "The user name for vcd API operations.",
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_PASSWORD", nil),
				Description: "The user password for vcd API operations.",
			},

			"org": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_ORG", nil),
				Description: "The vcd org for API operations",
			},

			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_URL", nil),
				Description: "The vcd url for vcd API operations.",
			},

			"maxRetryTimeout": &schema.Schema{
				Type:       schema.TypeInt,
				Optional:   true,
				Deprecated: "Deprecated. Use max_retry_timeout instead.",
			},

			"max_retry_timeout": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_MAX_RETRY_TIMEOUT", 60),
				Description: "Max num seconds to wait for successful response when operating on resources within vCloud (defaults to 60)",
			},

			"allow_unverified_ssl": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCD_ALLOW_UNVERIFIED_SSL", false),
				Description: "If set, VCDClient will permit unverifiable SSL certificates.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"vcd_network":         resourceVcdNetwork(),
			"vcd_vapp":            resourceVcdVApp(),
			"vcd_firewall_rules":  resourceVcdFirewallRules(),
			"vcd_dnat":            resourceVcdDNAT(),
			"vcd_snat":            resourceVcdSNAT(),
			"vcd_edgegateway_vpn": resourceVcdEdgeGatewayVpn(),
			"vcd_vapp_vm":         resourceVcdVAppVm(),
			"vcd_org":             resourceOrg(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	maxRetryTimeout := d.Get("max_retry_timeout").(int)

	// TODO: Deprecated, remove in next major release
	if v, ok := d.GetOk("maxRetryTimeout"); ok {
		maxRetryTimeout = v.(int)
	}

	config := Config{
		User:            d.Get("user").(string),
		Password:        d.Get("password").(string),
		Org:             d.Get("org").(string),
		Href:            d.Get("url").(string),
		MaxRetryTimeout: maxRetryTimeout,
		InsecureFlag:    d.Get("allow_unverified_ssl").(bool),
	}

	return config.Client()
}
