package css

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	css_logging "github.com/opentelekomcloud/gophertelekomcloud/openstack/css/v1/logs"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/fmterr"
)

func ResourceCssLoggingConfigurationV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCssLoggingConfigurationV1Create,
		ReadContext:   resourceCssLoggingConfigurationV1Read,
		UpdateContext: resourceCssLoggingConfigurationV1Update,
		DeleteContext: resourceCssLoggingConfigurationV1Delete,

		Importer: &schema.ResourceImporter{
			StateContext: resourceCssLoggingConfigurationV1Import,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
			},
			"agency": {
				Type:     schema.TypeString,
				Required: true,
			},
			"base_path": {
				Type:     schema.TypeString,
				Required: true,
			},
			"auto_backup": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"period": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceCssLoggingConfigurationV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := d.Get("cluster_id").(string)
	d.SetId(clusterID)

	opts := css_logging.EnableLogsOpts{
		Bucket:   d.Get("bucket").(string),
		Agency:   d.Get("agency").(string),
		BasePath: d.Get("base_path").(string),
	}

	if err := css_logging.EnableLogs(client, clusterID, opts); err != nil {
		return diag.FromErr(err)
	}

	if err := updateCssAutomaticLogBackup(d, client); err != nil {
		return diag.FromErr(err)
	}

	return resourceCssLoggingConfigurationV1Read(ctx, d, meta)
}

func resourceCssLoggingConfigurationV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := d.Id()
	loggingConfig, err := css_logging.GetConfiguration(client, clusterID)
	if err != nil {
		return fmterr.Errorf("error reading logs configuration: %s", err)
	}

	if !loggingConfig.LogSwitch {
		d.SetId("")
		return nil
	}

	d.Set("agency", loggingConfig.Agency)
	d.Set("bucket", loggingConfig.ObsBucket)
	d.Set("base_path", loggingConfig.BasePath)

	if loggingConfig.AutoEnable {
		autoBackupItem := map[string]interface{}{
			"period": loggingConfig.Period,
		}
		if err := d.Set("auto_backup", []interface{}{autoBackupItem}); err != nil {
			return fmterr.Errorf("failed to set auto_backup: %v", err)
		}
	} else {
		d.Set("auto_backup", nil)
	}

	return nil
}

func resourceCssLoggingConfigurationV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf("error creating CSS v1 client: %s", err)
	}

	if d.HasChanges("bucket", "agency", "base_path") {
		if err := updateCssLoggingConfiguration(client, d); err != nil {
			return fmterr.Errorf("error updating logging configuration: %s", err)
		}
	}

	if d.HasChange("auto_backup") {
		if err := updateCssAutomaticLogBackup(d, client); err != nil {
			return fmterr.Errorf("error updating automatic log backup for CSS cluster %s: %s", d.Id(), err)
		}
	}

	return resourceCssLoggingConfigurationV1Read(ctx, d, meta)
}

func updateCssLoggingConfiguration(client *golangsdk.ServiceClient, d *schema.ResourceData) error {
	opts := css_logging.UpdateLogConfigurationOpts{
		Bucket:   d.Get("bucket").(string),
		Agency:   d.Get("agency").(string),
		BasePath: d.Get("base_path").(string),
	}
	if err := css_logging.UpdateLogs(client, d.Id(), opts); err != nil {
		return fmt.Errorf("error updating CSS cluster logging: %w", err)
	}
	return nil
}

func updateCssAutomaticLogBackup(d *schema.ResourceData, client *golangsdk.ServiceClient) error {
	o, n := d.GetChange("auto_backup")
	oldList := o.([]interface{})
	newList := n.([]interface{})

	disable := func() error {
		if err := css_logging.DisableAutomaticBackups(client, d.Id()); err != nil {
			return fmt.Errorf("error disabling automatic log backup for CSS cluster %s: %s", d.Id(), err)
		}
		return nil
	}

	enable := func(period string) error {
		if err := css_logging.EnableAutomaticBackups(client, d.Id(), css_logging.EnableAutomaticBackupOpts{Period: period}); err != nil {
			return fmt.Errorf("error enabling automatic log backup for CSS cluster %s: %s", d.Id(), err)
		}
		return nil
	}

	switch {
	case len(newList) == 0 && len(oldList) > 0:
		return disable()
	case len(newList) > 0 && len(oldList) == 0:
		period := d.Get("auto_backup.0.period").(string)
		return enable(period)
	case len(newList) > 0 && len(oldList) > 0:
		if d.HasChange("auto_backup.0.period") {
			if err := disable(); err != nil {
				return err
			}
			period := d.Get("auto_backup.0.period").(string)
			return enable(period)
		}
	}
	return nil
}

func resourceCssLoggingConfigurationV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf("error creating CSS v1 client: %s", err)
	}

	if err := css_logging.DisableLogs(client, d.Id()); err != nil {
		return fmterr.Errorf("error disabling logging: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceCssLoggingConfigurationV1Import(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return nil, err
	}

	clusterID := d.Id()
	loggingConfig, err := css_logging.GetConfiguration(client, clusterID)
	if err != nil {
		return nil, err
	}

	if !loggingConfig.LogSwitch {
		return nil, fmt.Errorf("logging is disabled for cluster %s; cannot import resource", clusterID)
	}

	d.SetId(clusterID)
	d.Set("cluster_id", clusterID)
	d.Set("agency", loggingConfig.Agency)
	d.Set("bucket", loggingConfig.ObsBucket)
	d.Set("base_path", loggingConfig.BasePath)

	if loggingConfig.AutoEnable {
		autoBackupItem := map[string]interface{}{
			"period": loggingConfig.Period,
		}
		if err := d.Set("auto_backup", []interface{}{autoBackupItem}); err != nil {
			return nil, fmt.Errorf("failed to set auto_backup: %v", err)
		}
	} else {
		d.Set("auto_backup", nil)
	}

	return []*schema.ResourceData{d}, nil
}
