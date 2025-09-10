// package acceptance

// import (
// 	"fmt"
// 	"os"
// 	"testing"

// 	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
// 	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
// 	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
// )

// const bucketBasePath2 = "css/logs"

// func TestResourceCSSLoggingConfigurationV1_basic(t *testing.T) {
// 	name := fmt.Sprintf("css-%s", acctest.RandString(10))
// 	resourceName := "opentelekomcloud_css_logging_v1.logging"

// 	resource.ParallelTest(t, resource.TestCase{
// 		PreCheck: func() {
// 			common.TestAccPreCheck(t)
// 			// quotas.BookMany(t, sharedFlavorQuotas(t, 1, 100))
// 		},
// 		ProviderFactories: common.TestAccProviderFactories,
// 		// CheckDestroy:      testAccCheckCssClusterV1Destroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testResourceCSSLoggingV1Basic(name, bucketBasePath2),
// 				Check: resource.ComposeTestCheckFunc(
// 					resource.TestCheckResourceAttr(resourceName, "css/log", bucketBasePath2),
// 				),
// 			},
// 			// {
// 			// 	Config: testResourceCSSSnapshotConfigurationV1Updated(name, bucketBasePath),
// 			// 	Check: resource.ComposeTestCheckFunc(
// 			// 		resource.TestCheckResourceAttr(resourceName, "creation_policy.0.prefix", "snapshot"),
// 			// 		resource.TestCheckResourceAttr(resourceName, "creation_policy.0.period", "17:00 GMT+01:00"),
// 			// 		resource.TestCheckResourceAttr(resourceName, "creation_policy.0.keepday", "2"),
// 			// 	),
// 			// },
// 		},
// 	})
// }

// // func TestAccCheckCSSV1Validation(t *testing.T) {
// // 	name := fmt.Sprintf("css-%s", acctest.RandString(10))
// // 	resource.Test(t, resource.TestCase{
// // 		PreCheck:          func() { common.TestAccPreCheck(t) },
// // 		ProviderFactories: common.TestAccProviderFactories,
// // 		Steps: []resource.TestStep{
// // 			{
// // 				Config:      testResourceCSSSnapshotConfigurationV1Validation(name, bucketBasePath),
// // 				ExpectError: regexp.MustCompile(`Conflicting configuration.+`),
// // 			},
// // 		},
// // 	})
// // }

// func getOsAgency2() string {
// 	agency := os.Getenv("OS_CSS_OBS_AGENCY")
// 	if agency == "" {
// 		agency = "css_obs_agency"
// 	}
// 	return agency
// }

// var osAgency2 = getOsAgency2()

// func testResourceCSSLoggingV1Basic(name, bucketBasePath string) string {
// 	// relatedConfig := testAccCssClusterV1Basic(name)
// 	return fmt.Sprintf(`

// resource "opentelekomcloud_css_logging_v1" "logging2" {
//   cluster_id = opentelekomcloud_css_cluster_v1.cluster.id
//   bucket    = "asomogyiterraform"
//   agency    = %s
//   base_path = %s
//   auto_backup {
//     period = "00:00 GMT+08:00"
//   }
// }

// `, osAgency2, bucketBasePath)
// }

// // func testResourceCSSSnapshotConfigurationV1Updated(name, bucketBasePath string) string {
// // 	relatedConfig := testAccCssClusterV1Basic(name)
// // 	return fmt.Sprintf(`
// // %s

// // resource "opentelekomcloud_obs_bucket" "bucket" {
// //   bucket        = "tf-snap-testing"
// //   force_destroy = true
// // }

// // resource "opentelekomcloud_css_snapshot_configuration_v1" "config" {
// //   cluster_id = opentelekomcloud_css_cluster_v1.cluster.id
// //   configuration {
// //     bucket    = opentelekomcloud_obs_bucket.bucket.bucket
// //     agency    = "%s"
// //     base_path = "%s"
// //   }
// //   creation_policy {
// //     prefix      = "snapshot"
// //     period      = "17:00 GMT+01:00"
// //     keepday     = 2
// //     enable      = true
// //     delete_auto = true
// //   }
// // }
// // `, relatedConfig, osAgency, bucketBasePath)
// // }

// // func testResourceCSSSnapshotConfigurationV1Validation(name, bucketBasePath string) string {
// // 	relatedConfig := testAccCssClusterV1Basic(name)
// // 	return fmt.Sprintf(`
// // %s

// // resource "opentelekomcloud_obs_bucket" "bucket" {
// //   bucket        = "tf-snap-testing"
// //   force_destroy = true
// // }

// // resource "opentelekomcloud_css_snapshot_configuration_v1" "config" {
// //   cluster_id = opentelekomcloud_css_cluster_v1.cluster.id
// //   automatic  = true
// //   configuration {
// //     bucket    = opentelekomcloud_obs_bucket.bucket.bucket
// //     agency    = "%s"
// //     base_path = "%s"
// //   }
// // }
// // `, relatedConfig, osAgency, bucketBasePath)
// // }

package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/env"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/css/v1/clusters"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
)

const resourceCSSName = "opentelekomcloud_css_logging_v1.logging2"

func TestAccCSSLoggingV1_basic(t *testing.T) {
	var cluster clusters.Cluster
	// var vpc vpcs.Vpc
	// t.Parallel()
	// quotas.BookOne(t, quotas.Router)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { common.TestAccPreCheck(t) },
		ProviderFactories: common.TestAccProviderFactories,
		// CheckDestroy:      testAccCheckCSSLoggingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSSLoggingV1Basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCSSLoggingV1Exists(resourceCSSName, &cluster),
					resource.TestCheckResourceAttr(resourceCSSName, "cluster_id", "81eb5a57-2506-437b-bf16-a01ea71657b4"),
					resource.TestCheckResourceAttr(resourceCSSName, "bucket", "asomogyiterraform"),
					resource.TestCheckResourceAttr(resourceCSSName, "agency", "css_obs_agency"),
					resource.TestCheckResourceAttr(resourceCSSName, "base_path", "css/test_2/log"),
					resource.TestCheckResourceAttr(resourceCSSName, "auto_backup.period", "00:00 GMT+08:00"),
				),
			},
			// {
			// 	Config: testAccCSSLoggingV1Update,
			// 	Check: resource.ComposeTestCheckFunc(
			// 		// testAccCheckVpcV1Exists(resourceVPCName, &vpc),
			// 		resource.TestCheckResourceAttr(resourceVPCName, "name", "terraform_provider_test1"),
			// 		resource.TestCheckResourceAttr(resourceVPCName, "description", "simple description updated"),
			// 		resource.TestCheckResourceAttr(resourceVPCName, "shared", "false"),
			// 		resource.TestCheckResourceAttr(resourceVPCName, "tags.key", "value_update"),
			// 	),
			// },
		},
	})
}

// func getCssConfigurationV1ResourceFunc(cfg *cfg.Config, state *terraform.ResourceState) (interface{}, error) {
// 	c, err := cfg.CssV1Client(env.OS_REGION_NAME)
// 	if err != nil {
// 		return nil, fmt.Errorf("error creating APIG v2 client: %s", err)
// 	}
// 	configurations, err := pc.List(c, state.Primary.ID)
// 	if err != nil {
// 		return nil, fmt.Errorf("error retrieving OpenTelekomCloud CSS configuration: %s", err)
// 	}
// 	for _, template := range configurations.Templates {
// 		if template.Value != template.DefaultValue {
// 			return configurations, nil
// 		}
// 	}
// 	return nil, golangsdk.ErrDefault404{}
// }

// func testAccCheckCSSLoggingV1Destroy(s *terraform.State) error {
// 	config := common.TestAccProvider.Meta().(*cfg.Config)
// 	client, err := config.CssV1Client(env.OS_REGION_NAME)
// 	if err != nil {
// 		return fmt.Errorf("error enabling OpenTelekomCloud CSS logging: %w", err)
// 	}

// 	rs, ok := s.RootModule().Resources {
// 	if rs.Type != "opentelekomcloud_css_logging_v1" {
// 		continue
// 	}

// 	_, err := clusters.Get(client, rs.Primary.).Extract()
// 	if err == nil {
// 		return fmt.Errorf("css still exists")
// 	}

// 	return nil
// }

func testAccCheckCSSLoggingV1Exists(n string, cluster *clusters.Cluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		config := common.TestAccProvider.Meta().(*cfg.Config)
		client, err := config.CssV1Client(env.OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("error creating CSSv1 client: %w", err)
		}

		found, err := clusters.Get(client, rs.Primary.ID)
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("cluster not found")
		}

		*cluster = *found

		return nil
	}
}

// func testAccCheckVpcV1Exists(n string, vpc *vpcs.Vpc) resource.TestCheckFunc {
// 	return func(s *terraform.State) error {
// 		rs, ok := s.RootModule().Resources[n]
// 		if !ok {
// 			return fmt.Errorf("not found: %s", n)
// 		}

// 		if rs.Primary.ID == "" {
// 			return fmt.Errorf("no ID is set")
// 		}

// 		config := common.TestAccProvider.Meta().(*cfg.Config)
// 		client, err := config.NetworkingV1Client(env.OS_REGION_NAME)
// 		if err != nil {
// 			return fmt.Errorf("error creating OpenTelekomCloud NetworkingV1 client: %w", err)
// 		}

// 		found, err := vpcs.Get(client, rs.Primary.ID).Extract()
// 		if err != nil {
// 			return err
// 		}

// 		if found.ID != rs.Primary.ID {
// 			return fmt.Errorf("vpc not found")
// 		}

// 		*vpc = *found

// 		return nil
// 	}
// }

const testAccCSSLoggingV1Basic = `
resource "opentelekomcloud_css_logging_v1" "logging2" {
  cluster_id = "81eb5a57-2506-437b-bf16-a01ea71657b4"
  bucket    = "asomogyiterraform"
  agency    = "css_obs_agency"
  base_path = "css/test_2/log"
  auto_backup {
    period = "00:00 GMT+08:00"
  }
}
`

// const testAccCSSLoggingV1Basic = `
// resource "opentelekomcloud_vpc_v1" "vpc_1" {
//   name        = "terraform_provider_test"
//   description = "simple description"
//   cidr        = "192.168.0.0/16"
//   shared      = true

//   tags = {
//     foo = "bar"
//     key = "value"
//   }
// }
// `

const testAccCSSLoggingV1Update = `
resource "opentelekomcloud_vpc_v1" "vpc_1" {
  name        = "terraform_provider_test1"
  description = "simple description updated"
  cidr        = "192.168.0.0/16"
  shared      = false

  tags = {
    foo = "bar"
    key = "value_update"
  }
}
`
