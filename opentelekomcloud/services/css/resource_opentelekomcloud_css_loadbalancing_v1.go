package css

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/css/v1/load_balancer"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/fmterr"
)

func ResourceCSSLoadBalancingConfigurationV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: createCSSLoadBalancingConfigurationV1,
		ReadContext:   readCSSLoadBalancingConfigurationV1,
		UpdateContext: updateCSSLoadBalancingConfigurationV1,
		DeleteContext: deleteCSSLoadBalancingConfigurationV1,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"agency": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"elb_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"listener": {
				Type:     schema.TypeList,
				Optional: true,
				// ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type: schema.TypeString,
							// Required: true,
							ForceNew: true,
							Required: true,
						},
						"protocol_port": {
							Type: schema.TypeInt,
							// Required: true,
							ForceNew: true,
							Required: true,
						},
						"server_cert_id": {
							Type: schema.TypeString,
							// ForceNew: true,
							Optional: true,
						},
						"ca_cert_id": {
							Type: schema.TypeString,
							// ForceNew: true,
							Optional: true,
						},
						"type": {
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func createCSSLoadBalancingConfigurationV1(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(clientError, err)
	}

	clusterID := d.Get("cluster_id").(string)
	agency := d.Get("agency").(string)
	elbId := d.Get("elb_id").(string)
	protocol := d.Get("listener.0.protocol").(string)
	protocolPort := d.Get("listener.0.protocol_port").(int)
	serverCertId := d.Get("listener.0.server_cert_id").(string)
	caCertId := d.Get("listener.0.ca_cert_id").(string)
	listenerType := d.Get("listener.0.type").(string)
	d.SetId(clusterID)
	d.Set("agency", agency)
	d.Set("elb_id", elbId)
	d.Set("listener.0.protocol", protocol)
	d.Set("listener.0.protocol_port", protocolPort)
	d.Set("listener.0.server_cert_id", serverCertId)
	d.Set("listener.0.ca_cert_id", caCertId)
	d.Set("listener.0.type", listenerType)

	if err := createLoadBalancingConfiguration(client, d); err != nil {
		return diag.FromErr(err)
	}

	return readCSSLoadBalancingConfigurationV1(ctx, d, meta)
}

func readCSSLoadBalancingConfigurationV1(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(clientError, err)
	}
	clusterID := d.Id()

	info, err := load_balancer.Get(client, clusterID)

	if err != nil {
		return fmterr.Errorf("error retrieving the CSS cluster's load balancing configuration")
	}

	mErr := multierror.Append(
		d.Set("agency", info.Agency),
		d.Set("elb_id", info.LoadBalancer.Id),
		d.Set("listener.0.protocol", info.Listener.Protocol),
		d.Set("listener.0.protocol_port", info.Listener.ProtocolPort),
		d.Set("listener.0.ca_cert_id", info.CacertId),
		d.Set("listener.0.server_cert_id", info.ServerCertId),
	)

	if err := mErr.ErrorOrNil(); err != nil {
		fmterr.Errorf("error setting css load configuration fields: %w", err)
	}

	return nil
}

func updateCSSLoadBalancingConfigurationV1(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(clientError, err)
	}

	if d.HasChange("listener") {
		err := updateCSSELBlistener(d, client)
		if err != nil {
			return fmterr.Errorf("error updating the load balancing listener of CSS cluster %s: %s", d.Id(), err)
		}
	}

	return readCSSLoggingConfigurationV1(ctx, d, meta)
}

func updateCSSELBlistener(d *schema.ResourceData, client *golangsdk.ServiceClient) error {
	o, n := d.GetChange("listener")
	oValue := o.([]interface{})
	nValue := n.([]interface{})

	switch len(nValue) - len(oValue) {
	case -1:
		fmt.Print("CASE -1")
		_, err := load_balancer.DisableLoadBalancer(client, d.Id())
		if err != nil {
			return fmt.Errorf("error disabling CSS cluster's load balancing: %s, err: %s", d.Id(), err)
		}
		if d.Get("elb_id") != "" {
			_, err = load_balancer.EnableLoadBalancer(client, d.Id(), load_balancer.EnableLoadBalancerOpts{
				ElbId:  d.Get("elb_id").(string),
				Agency: d.Get("agency").(string),
			})
			if err != nil {
				return fmt.Errorf("error enabling CSS cluster's load balancing: %s, err: %s", d.Id(), err)
			}
		}
		load_balancer.WaitForListenerStatus(client, d.Get("elb_id").(string), 2)

	case 1:
		fmt.Print("CASE 1")
		details, err := load_balancer.Get(client, d.Id())

		if err != nil {
			return fmt.Errorf("error getting the information about the CSS cluster's load balancing: %s, err: %s", d.Id(), err)
		}

		if details.Enabled == false {
			_, err = load_balancer.EnableLoadBalancer(client, d.Id(), load_balancer.EnableLoadBalancerOpts{
				ElbId:  d.Get("elb_id").(string),
				Agency: d.Get("agency").(string),
			})
			if err != nil {
				return fmt.Errorf("error enabling CSS cluster's load balancing: %s, err: %s", d.Id(), err)
			}
		}
		load_balancer.WaitForListenerStatus(client, d.Get("elb_id").(string), 2)

		if strings.ToUpper(d.Get("listener.0.protocol").(string)) == "HTTP" {
			_, err = load_balancer.ConfigureListener(client, d.Id(), load_balancer.CongigureListenerOpts{
				Protocol:     d.Get("listener.0.protocol").(string),
				ProtocolPort: d.Get("listener.0.protocol_port").(int),
			})
			if err != nil {
				return fmt.Errorf("error enabling the load balancing's listener: %s, err: %s", d.Id(), err)
			}
		}

		if strings.ToUpper(d.Get("listener.0.protocol").(string)) == "HTTPS" {
			_, err = load_balancer.ConfigureListener(client, d.Id(), load_balancer.CongigureListenerOpts{
				Protocol:     d.Get("listener.0.protocol").(string),
				ProtocolPort: d.Get("listener.0.protocol_port").(int),
				ServerCertId: d.Get("listener.0.server_cert_id").(string),
				CaCertId:     d.Get("listener.0.ca_cert_id").(string),
				Type:         d.Get("listener.0.type").(string),
			})
			if err != nil {
				return fmt.Errorf("error enabling the load balancing's listener: %s, err: %s", d.Id(), err)
			}
		}

	case 0:
		fmt.Print("CASE 0")

		if d.HasChanges("listener.0.server_cert_id", "listener.0.ca_cert_id") {
			details, err := load_balancer.Get(client, d.Id())
			if err != nil {
				return fmt.Errorf("error getting the details of the CSS load balancer: %s, err: %s", d.Id(), err)
			}

			listenerId := details.Listener.Id

			updateListenerOpts := load_balancer.UpdateListenerOpts{}
			updateListenerOpts.Listener.ServerCertId = d.Get("listener.0.server_cert_id").(string)
			updateListenerOpts.Listener.CaCertId = d.Get("listener.0.ca_cert_id").(string)

			_, err = load_balancer.UpdateListener(client, d.Id(), listenerId, updateListenerOpts)
			if err != nil {
				return fmt.Errorf("error updating the load balancer's listener: %s, err: %s", d.Id(), err)
			}
		}
	}

	return nil
}

func deleteCSSLoadBalancingConfigurationV1(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(clientError, err)
	}
	clusterID := d.Id()

	if _, err := load_balancer.DisableLoadBalancer(client, clusterID); err != nil {
		return fmterr.Errorf("error disabling css load balancing: %w", err)
	}

	return nil
}

func createLoadBalancingConfiguration(client *golangsdk.ServiceClient, d *schema.ResourceData) error {
	enableELBOpts := load_balancer.EnableLoadBalancerOpts{
		ElbId:  d.Get("elb_id").(string),
		Agency: d.Get("agency").(string),
	}

	configuringListenerOpts := load_balancer.CongigureListenerOpts{
		Protocol:     d.Get("listener.0.protocol").(string),
		ProtocolPort: d.Get("listener.0.protocol_port").(int),
		ServerCertId: d.Get("listener.0.server_cert_id").(string),
		CaCertId:     d.Get("listener.0.ca_cert_id").(string),
		Type:         d.Get("listener.0.type").(string),
	}

	_, err := load_balancer.EnableLoadBalancer(client, d.Id(), enableELBOpts)
	if err != nil {
		return fmt.Errorf("error enabling css cluster load balancing: %w", err)
	}
	load_balancer.WaitForListenerStatus(client, d.Get("elb_id").(string), 2)

	if d.Get("listener.0.protocol") != "" {
		_, err = load_balancer.ConfigureListener(client, d.Id(), configuringListenerOpts)
		if err != nil {
			return fmt.Errorf("error configuring the listener for css cluster load balancing: %w", err)
		}
	}

	return err
}
