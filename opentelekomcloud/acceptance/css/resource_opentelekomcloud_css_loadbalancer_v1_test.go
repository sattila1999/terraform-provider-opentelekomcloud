package acceptance

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/css/v1/load_balancer"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/env"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

// getCssLoggingConfigurationFunc loads the CSS logging config from the remote API
func getCssLoadBalancerFunc(conf *cfg.Config, state *terraform.ResourceState) (interface{}, error) {
	client, err := conf.CssV1Client(env.OS_REGION_NAME)
	if err != nil {
		return nil, fmt.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := os.Getenv("OS_CSS_CLUSTER_ID")
	if clusterID == "" {
		return nil, fmt.Errorf("environment variable OS_CSS_CLUSTER_ID is not set")
	}

	return load_balancer.Get(client, clusterID)
}

// TestAccCssLoggingConfiguration_basic acceptance test for CSS logging config
func TestAccCssLoadBalancerConfiguration_basic(t *testing.T) {
	clusterID := os.Getenv("OS_CSS_CLUSTER_ID")
	if clusterID == "" {
		t.Skip("OS_CSS_CLUSTER_ID env var is not set")
	}

	var obj load_balancer.LoadBalancerResp
	rName := "opentelekomcloud_css_loadbalancer_v1.css_lb"
	rc := common.InitResourceCheck(
		rName,
		&obj,
		getCssLoadBalancerFunc,
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { common.TestAccPreCheck(t) },
		ProviderFactories: common.TestAccProviderFactories,
		CheckDestroy:      checkCssLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testCssLoadBalancerV1_basic(clusterID),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "agency", "css_upgrade_agency"),
				),
			},
			{
				Config: testCssLoadBalancerV1_update(clusterID),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "agency", "css_upgrade_agency"),
				),
			},
			{
				Config: testCssLoadBalancerV1_update2(clusterID),
				Check: resource.ComposeTestCheckFunc(
					rc.CheckResourceExists(),
					resource.TestCheckResourceAttr(rName, "agency", "css_upgrade_agency"),
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

// testCssLoadBalancerV1_basic returns Terraform config string
func testCssLoadBalancerV1_basic(clusterID string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_css_loadbalancer_v1" "css_lb" {
  cluster_id = "%s"
  elb_id     = "9f15d81e-7b3e-430d-828e-5a076b990145"
  agency     = "css_upgrade_agency"
  listener {
    protocol       = "HTTPS"
    protocol_port  = 443
    server_cert_id = "786b11e555e5459a8ed86f678eb7ef21"
  }
}
`, clusterID)
}

// testCssLoadBalancerV1_update returns Terraform config string
func testCssLoadBalancerV1_update(clusterID string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_css_loadbalancer_v1" "css_lb" {
  cluster_id = "%s"
  elb_id     = "9f15d81e-7b3e-430d-828e-5a076b990145"
  agency     = "css_upgrade_agency"
}
`, clusterID)
}

func testCssLoadBalancerV1_update2(clusterID string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_css_loadbalancer_v1" "css_lb" {
  cluster_id = "%s"
  elb_id     = "9f15d81e-7b3e-430d-828e-5a076b990145"
  agency     = "css_upgrade_agency"
  listener {
    protocol       = "HTTPS"
    protocol_port  = 443
    server_cert_id = "786b11e555e5459a8ed86f678eb7ef21"
  }
}
`, clusterID)
}

// checkCssLoggingConfigDestroy verifies the CSS logging configuration was destroyed
func checkCssLoadBalancerDestroy(s *terraform.State) error {
	config := common.TestAccProvider.Meta().(*cfg.Config)

	client, err := config.CssV1Client(env.OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("error creating CSS v1 client: %s", err)
	}

	clusterID := os.Getenv("OS_CSS_CLUSTER_ID")
	if clusterID == "" {
		return fmt.Errorf("environment variable OS_CSS_CLUSTER_ID is not set")
	}

	load_balancer_config, err := load_balancer.Get(client, clusterID)
	if err != nil {
		// Assume resource is gone if error occurs here
		return nil
	}

	if load_balancer_config.Enabled {
		return fmt.Errorf("load_balancer is still enabled for cluster %s", clusterID)
	}

	return nil
}
