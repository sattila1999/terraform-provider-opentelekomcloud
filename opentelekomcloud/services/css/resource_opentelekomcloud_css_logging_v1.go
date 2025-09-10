package css

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/css/v1/logs"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/fmterr"
)

func ResourceCSSLoggingConfigurationV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: createCSSLoggingConfigurationV1,
		ReadContext:   readCSSLoggingConfigurationV1,
		UpdateContext: updateCSSLoggingConfigurationV1,
		DeleteContext: deleteCSSLoggingConfigurationV1,

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

func createCSSLoggingConfigurationV1(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(clientError, err)
	}

	clusterID := d.Get("cluster_id").(string)
	bucket := d.Get("bucket").(string)
	agency := d.Get("agency").(string)
	base_path := d.Get("base_path").(string)
	d.SetId(clusterID)
	d.Set("bucket", bucket)
	d.Set("agency", agency)
	d.Set("base_path", base_path)

	if err := createLoggingConfiguration(client, d); err != nil {
		return diag.FromErr(err)
	}

	err = updateCssAutomaticLogBackup(d, client)
	if err != nil {
		return diag.FromErr(err)
	}

	return readCSSLoggingConfigurationV1(ctx, d, meta)
}

func readCSSLoggingConfigurationV1(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(clientError, err)
	}
	clusterID := d.Id()

	info, err := logs.GetConfiguration(client, clusterID)

	if err != nil {
		return fmterr.Errorf("error retrieving CSS cluster logging configuration")
	}

	// "id": info.ID,
	// "clusterId": info,
	// "obsBucket": "asomogyi-test",
	// "agency": "css_obs_agency",
	// "updateAt": 1747817633159,
	// "basePath": "css/log",
	// "autoEnable": false,
	// "period": "00:00 GMT+08:00",
	// "logSwitch": false,

	mErr := multierror.Append(
		d.Set("cluster_id", info.ClusterID),
		d.Set("bucket", info.ObsBucket),
		d.Set("agency", info.Agency),
		d.Set("base_path", info.BasePath),
		d.Set("auto_backup.0.period", info.Period),
	)

	if err := mErr.ErrorOrNil(); err != nil {
		fmterr.Errorf("error setting css logging configuration fields: %w", err)
	}

	return nil
}

func updateCSSLoggingConfigurationV1(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(clientError, err)
	}

	if d.HasChanges("bucket", "agency", "base_path") {
		if err := updateLoggingConfiguration(client, d); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("auto_backup") {
		err := updateCssAutomaticLogBackup(d, client)
		if err != nil {
			return fmterr.Errorf("error updating automatic log backup for CSS cluster %s: %s", d.Id(), err)
		}
	}

	return readCSSLoggingConfigurationV1(ctx, d, meta)
}

func deleteCSSLoggingConfigurationV1(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	client, err := config.CssV1Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(clientError, err)
	}
	clusterID := d.Id()

	if err := logs.DisableLogs(client, clusterID); err != nil {
		return fmterr.Errorf("error disabling css logs: %w", err)
	}

	return nil
}

func updateLoggingConfiguration(client *golangsdk.ServiceClient, d *schema.ResourceData) error {
	opts := logs.UpdateLogConfigurationOpts{
		Bucket:   d.Get("bucket").(string),
		Agency:   d.Get("agency").(string),
		BasePath: d.Get("base_path").(string),
	}
	err := logs.UpdateLogs(client, d.Id(), opts)
	if err != nil {
		return fmt.Errorf("error enabling css cluster logging: %w", err)
	}
	return nil
}

func createLoggingConfiguration(client *golangsdk.ServiceClient, d *schema.ResourceData) error {
	opts := logs.EnableLogsOpts{
		Bucket:   d.Get("bucket").(string),
		Agency:   d.Get("agency").(string),
		BasePath: d.Get("base_path").(string),
	}
	err := logs.EnableLogs(client, d.Id(), opts)
	if err != nil {
		return fmt.Errorf("error enabling css cluster logging: %w", err)
	}
	return nil
}

func updateCssAutomaticLogBackup(d *schema.ResourceData, client *golangsdk.ServiceClient) error {
	o, n := d.GetChange("auto_backup")
	oValue := o.([]interface{})
	nValue := n.([]interface{})

	switch len(nValue) - len(oValue) {
	case -1: // disable automtic log backup policy
		err := logs.DisableAutomaticBackups(client, d.Id())
		if err != nil {
			return fmt.Errorf("error disabling CSS cluster's automatic log backup policy: %s, err: %s", d.Id(), err)
		}

	case 1:
		// enable automatic log backup policy
		err := logs.EnableAutomaticBackups(client, d.Id(), logs.EnableAutomaticBackupOpts{
			Period: d.Get("auto_backup.0.period").(string),
		})

		if err != nil {
			return fmt.Errorf("error enabling CSS cluster's automatic log backup policy: %s, err: %s", d.Id(), err)
		}

	case 0:
		// disable and enable automatic log backup policy based on changes
		if d.HasChanges("auto_backup.0.period") {
			err := logs.DisableAutomaticBackups(client, d.Id())
			if err != nil {
				return fmt.Errorf("error disabling CSS cluster's automatic log backup policy: %s, err: %s", d.Id(), err)
			}
			err = logs.EnableAutomaticBackups(client, d.Id(), logs.EnableAutomaticBackupOpts{
				Period: d.Get("auto_backup.0.period").(string),
			})

			if err != nil {
				return fmt.Errorf("error updating CSS cluster's automatic log backup policy: %s, err: %s", d.Id(), err)
			}
		}
	}

	return nil
}
