package css

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/css/v1/load_balancer"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

const timeoutSeconds = 300

// ResourceCssLoadBalancerV1 defines the Terraform resource for managing
// CSS (Cloud Search Service) Load Balancer configuration.
func ResourceCssLoadBalancerV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCssLoadBalancerCreate,
		ReadContext:   resourceCssLoadBalancerRead,
		UpdateContext: resourceCssLoadBalancerUpdate,
		DeleteContext: resourceCssLoadBalancerDelete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceCssLoadBalancerV1Import,
		},

		// Define the schema for the resource
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"elb_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"agency": {
				Type:     schema.TypeString,
				Required: true,
			},
			"listener": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"server_cert_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ca_cert_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

// resourceCssLoadBalancerCreate enables ELB for a CSS cluster
// and configures a listener if provided.
func resourceCssLoadBalancerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := d.Get("cluster_id").(string)
	elbID := d.Get("elb_id").(string)
	agency := d.Get("agency").(string)

	// Set resource ID to cluster ID
	d.SetId(clusterID)

	// Enable load balancer on cluster
	if _, err := load_balancer.EnableLoadBalancer(client, clusterID, load_balancer.EnableLoadBalancerOpts{
		ElbId:  elbID,
		Agency: agency,
	}); err != nil {
		return diag.Errorf("failed to enable load balancer: %s", err)
	}

	// Configure listener if specified
	if opts := extractListenerOpts(d); opts != nil {
		if _, err := load_balancer.ConfigureListener(client, clusterID, *opts); err != nil {
			return diag.Errorf("failed to configure listener: %s", err)
		}
		if err := load_balancer.WaitForListenerStatus(client, clusterID, timeoutSeconds); err != nil {
			return diag.Errorf("error waiting for listener to be active: %s", err)
		}
	}

	return resourceCssLoadBalancerRead(ctx, d, meta)
}

// resourceCssLoadBalancerRead fetches current CSS LB configuration
func resourceCssLoadBalancerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := d.Id()

	details, err := load_balancer.Get(client, clusterID)
	if err != nil {
		return diag.Errorf("failed to get load balancer details: %s", err)
	}

	// If LB is disabled, remove resource from state
	if details == nil || !details.Enabled {
		d.SetId("")
		return nil
	}

	// Set resource attributes
	d.Set("elb_id", details.LoadBalancer.Id)
	d.Set("agency", details.Agency)
	d.Set("cluster_id", clusterID)

	// Sync listener configuration
	if details.Listener.Id != "" {
		listener := map[string]interface{}{
			"protocol":       details.Listener.Protocol,
			"protocol_port":  details.Listener.ProtocolPort,
			"server_cert_id": details.ServerCertId,
			"ca_cert_id":     details.CacertId,
		}
		d.Set("listener", []interface{}{listener})
	} else {
		d.Set("listener", nil)
	}

	return nil
}

// resourceCssLoadBalancerUpdate updates listener or ELB ID if changed.
func resourceCssLoadBalancerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating CSS v1 client: %s", err)
	}

	// Apply listener or ELB ID changes
	if d.HasChange("listener") || d.HasChange("elb_id") {
		if err := updateListener(d, client); err != nil {
			return diag.Errorf("failed to update listener: %s", err)
		}
	}

	return resourceCssLoadBalancerRead(ctx, d, meta)
}

// resourceCssLoadBalancerDelete disables ELB for the CSS cluster.
func resourceCssLoadBalancerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Create CSS v1 client
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return diag.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := d.Id()

	// Disable LB
	if _, err := load_balancer.DisableLoadBalancer(client, clusterID); err != nil {
		return diag.Errorf("failed to disable load balancer: %s", err)
	}

	// Remove from state
	d.SetId("")
	return nil
}

// extractListenerOpts extracts listener options from schema data.
func extractListenerOpts(d *schema.ResourceData) *load_balancer.CongigureListenerOpts {
	list := d.Get("listener").([]interface{})
	if len(list) == 0 {
		return nil
	}

	m := list[0].(map[string]interface{})
	opts := load_balancer.CongigureListenerOpts{
		Protocol:     m["protocol"].(string),
		ProtocolPort: m["protocol_port"].(int),
	}

	// For HTTPS listeners, set certs and type
	if strings.ToUpper(opts.Protocol) == "HTTPS" {
		if v, ok := m["server_cert_id"].(string); ok && v != "" {
			opts.ServerCertId = v
		}
		if v, ok := m["ca_cert_id"].(string); ok && v != "" {
			opts.CaCertId = v
		}
		if v, ok := m["type"].(string); ok && v != "" {
			opts.Type = v
		}
	}

	return &opts
}

// updateListener handles all listener-related updates, including
// add, remove, modify.
func updateListener(d *schema.ResourceData, client *golangsdk.ServiceClient) error {
	clusterID := d.Id()
	elbID := d.Get("elb_id").(string)
	agency := d.Get("agency").(string)

	oldList, newList := d.GetChange("listener")
	oldLen := len(oldList.([]interface{}))
	newLen := len(newList.([]interface{}))

	switch {
	case oldLen > 0 && newLen == 0:
		// Listener removed: disable and re-enable ELB
		if _, err := load_balancer.DisableLoadBalancer(client, clusterID); err != nil {
			return err
		}
		_, err := load_balancer.EnableLoadBalancer(client, clusterID, load_balancer.EnableLoadBalancerOpts{
			ElbId:  elbID,
			Agency: agency,
		})
		return err

	case newLen > 0:
		// Listener added or updated
		opts := extractListenerOpts(d)
		if opts != nil {
			if _, err := load_balancer.ConfigureListener(client, clusterID, *opts); err != nil {
				return err
			}
			if err := load_balancer.WaitForListenerStatus(client, clusterID, timeoutSeconds); err != nil {
				return err
			}
		}

		// If ELB ID changed, re-enable LB with new ID
		if d.HasChange("elb_id") {
			if _, err := load_balancer.EnableLoadBalancer(client, clusterID, load_balancer.EnableLoadBalancerOpts{
				ElbId:  elbID,
				Agency: agency,
			}); err != nil {
				return err
			}
		}
		return nil

	default:
		// Only ELB ID changed (no listener changes)
		if d.HasChange("elb_id") {
			_, err := load_balancer.EnableLoadBalancer(client, clusterID, load_balancer.EnableLoadBalancerOpts{
				ElbId:  elbID,
				Agency: agency,
			})
			return err
		}
		return nil
	}
}

// resourceCssLoadBalancerV1Import allows importing
// existing LB configuration into Terraform state.
func resourceCssLoadBalancerV1Import(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return nil, err
	}

	clusterID := d.Id()

	details, err := load_balancer.Get(client, clusterID)
	if err != nil {
		return nil, err
	}

	// If LB is disabled, import not possible
	if !details.Enabled {
		return nil, fmt.Errorf("load balancer is disabled for cluster %s; cannot import resource", clusterID)
	}

	// Sync attributes into state
	d.Set("elb_id", details.LoadBalancer.Id)
	d.Set("agency", details.Agency)
	d.Set("cluster_id", clusterID)

	if details.Listener.Id != "" {
		listener := map[string]interface{}{
			"protocol":       details.Listener.Protocol,
			"protocol_port":  details.Listener.ProtocolPort,
			"server_cert_id": details.ServerCertId,
			"ca_cert_id":     details.CacertId,
		}
		if err := d.Set("listener", []interface{}{listener}); err != nil {
			return nil, fmt.Errorf("failed to set listener: %v", err)
		}
	}

	return []*schema.ResourceData{d}, nil
}
