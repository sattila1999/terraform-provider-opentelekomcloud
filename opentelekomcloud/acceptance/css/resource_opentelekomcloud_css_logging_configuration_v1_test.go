package acceptance

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/css/v1/logs"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/env"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

var css_obs_agency = "css_obs_agency"

// getCssLoggingConfigurationFunc loads the CSS logging config from the remote API
func getCssLoggingConfigurationFunc(conf *cfg.Config, state *terraform.ResourceState) (interface{}, error) {
	client, err := conf.CssV1Client(env.OS_REGION_NAME)
	if err != nil {
		return nil, fmt.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := os.Getenv("OS_CSS_CLUSTER_ID")
	if clusterID == "" {
		return nil, fmt.Errorf("environment variable OS_CSS_CLUSTER_ID is not set")
	}

	return logs.GetConfiguration(client, clusterID)
}

// TestAccCssLoggingConfiguration_basic acceptance test for CSS logging config
func TestAccCssLoggingConfiguration_basic(t *testing.T) {
	clusterID := os.Getenv("OS_CSS_CLUSTER_ID")
	if clusterID == "" {
		t.Skip("OS_CSS_CLUSTER_ID env var is not set")
	}

	var obj logs.LogConfiguration
	rName := "opentelekomcloud_css_logging_configuration_v1.config"
	rc := common.InitResourceCheck(
		rName,
		&obj,
		getCssLoggingConfigurationFunc,
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { common.TestAccPreCheck(t) },
		ProviderFactories: common.TestAccProviderFactories,
		CheckDestroy:      checkCssLoggingConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testCssLoggingConfigurationV1_basic(clusterID),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "agency", css_obs_agency),
					resource.TestCheckResourceAttr(rName, "base_path", "css/test_2/log"),
					resource.TestCheckResourceAttr(rName, "bucket", "asomogyiterraform"),
					resource.TestCheckResourceAttr(rName, "auto_backup.0.period", "00:00 GMT+08:00"),
				),
			},
			{
				Config: testCssLoggingConfigurationV1_update(clusterID),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "agency", css_obs_agency),
					resource.TestCheckResourceAttr(rName, "base_path", "css/test_3/log"),
					resource.TestCheckResourceAttr(rName, "bucket", "asomogyiterraform"),
					resource.TestCheckResourceAttr(rName, "auto_backup.#", "0"),
				),
			},
			{
				Config: testCssLoggingConfigurationV1_update2(clusterID),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "agency", css_obs_agency),
					resource.TestCheckResourceAttr(rName, "base_path", "css/test_3/log"),
					resource.TestCheckResourceAttr(rName, "bucket", "asomogyiterraform"),
					resource.TestCheckResourceAttr(rName, "auto_backup.0.period", "00:00 GMT+07:00"),
				),
			},
			{
				ResourceName:      rName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// testCssLoggingConfigurationV1_basic returns Terraform config string
func testCssLoggingConfigurationV1_basic(clusterID string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_css_logging_configuration_v1" "config" {
  cluster_id = "%s"
  bucket     = "asomogyiterraform"
  agency     = "%s"
  base_path  = "css/test_2/log"
  auto_backup {
    period = "00:00 GMT+08:00"
  }
}
`, clusterID, css_obs_agency)
}

func testCssLoggingConfigurationV1_update(clusterID string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_css_logging_configuration_v1" "config" {
  cluster_id = "%s"
  bucket     = "asomogyiterraform"
  agency     = "%s"
  base_path  = "css/test_3/log"
}
`, clusterID, css_obs_agency)
}

func testCssLoggingConfigurationV1_update2(clusterID string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_css_logging_configuration_v1" "config" {
  cluster_id = "%s"
  bucket     = "asomogyiterraform"
  agency     = "%s"
  base_path  = "css/test_3/log"
  auto_backup {
    period = "00:00 GMT+07:00"
  }
}
`, clusterID, css_obs_agency)
}

// checkCssLoggingConfigDestroy verifies the CSS logging configuration was destroyed
func checkCssLoggingConfigDestroy(s *terraform.State) error {
	config := common.TestAccProvider.Meta().(*cfg.Config)

	client, err := config.CssV1Client(env.OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := os.Getenv("OS_CSS_CLUSTER_ID")
	if clusterID == "" {
		return fmt.Errorf("environment variable OS_CSS_CLUSTER_ID is not set")
	}

	loggingConfig, err := logs.GetConfiguration(client, clusterID)
	if err != nil {
		// Assume resource is gone if error occurs here
		return nil
	}

	if loggingConfig.LogSwitch {
		return fmt.Errorf("logging still enabled for cluster %s", clusterID)
	}

	return nil
}
