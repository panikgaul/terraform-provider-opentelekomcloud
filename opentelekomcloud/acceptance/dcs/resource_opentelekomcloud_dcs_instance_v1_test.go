package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/dcs/v1/instances"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/env"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

func TestAccDcsInstancesV1_depr(t *testing.T) {
	var instance instances.Instance
	var instanceName = fmt.Sprintf("dcs_instance_%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckDcs(t) },
		Providers:    common.TestAccProviders,
		CheckDestroy: testAccCheckDcsV1InstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDcsV1Instance_depr(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDcsV1InstanceExists("opentelekomcloud_dcs_instance_v1.instance_1", instance),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_dcs_instance_v1.instance_1", "name", instanceName),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_dcs_instance_v1.instance_1", "engine", "Redis"),
				),
			},
		},
	})
}

func TestAccDcsInstancesV1_basic(t *testing.T) {
	var instance instances.Instance
	var instanceName = fmt.Sprintf("dcs_instance_%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { common.TestAccPreCheck(t) },
		Providers:    common.TestAccProviders,
		CheckDestroy: testAccCheckDcsV1InstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDcsV1Instance_basic(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDcsV1InstanceExists("opentelekomcloud_dcs_instance_v1.instance_1", instance),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_dcs_instance_v1.instance_1", "name", instanceName),
					resource.TestCheckResourceAttr(
						"opentelekomcloud_dcs_instance_v1.instance_1", "engine", "Redis"),
				),
			},
			{
				Config: testAccDcsV1Instance_updated(instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("opentelekomcloud_dcs_instance_v1.instance_1", "backup_policy.0.begin_at", "01:00-02:00"),
					resource.TestCheckResourceAttr("opentelekomcloud_dcs_instance_v1.instance_1", "backup_policy.0.save_days", "2"),
					resource.TestCheckResourceAttr("opentelekomcloud_dcs_instance_v1.instance_1", "backup_policy.0.backup_at.#", "3"),
				),
			},
			{
				Config: testAccDcsV1Instance_single(instanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDcsV1InstanceExists("opentelekomcloud_dcs_instance_v1.instance_1", instance),
					resource.TestCheckResourceAttr("opentelekomcloud_dcs_instance_v1.instance_1", "name", instanceName),
					resource.TestCheckResourceAttr("opentelekomcloud_dcs_instance_v1.instance_1", "engine", "Redis"),
					resource.TestCheckResourceAttr("opentelekomcloud_dcs_instance_v1.instance_1", "resource_spec_code", "dcs.single_node"),
				),
			},
		},
	})
}

func testAccCheckDcsV1InstanceDestroy(s *terraform.State) error {
	config := common.TestAccProvider.Meta().(*cfg.Config)
	dcsClient, err := config.DcsV1Client(env.OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating instance client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opentelekomcloud_dcs_instance_v1" {
			continue
		}

		_, err := instances.Get(dcsClient, rs.Primary.ID).Extract()
		if err == nil {
			return fmt.Errorf("The Dcs instance still exists.")
		}
	}
	return nil
}

func testAccCheckDcsV1InstanceExists(n string, instance instances.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config := common.TestAccProvider.Meta().(*cfg.Config)
		dcsClient, err := config.DcsV1Client(env.OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("Error creating instance client: %s", err)
		}

		v, err := instances.Get(dcsClient, rs.Primary.ID).Extract()
		if err != nil {
			return fmt.Errorf("Error getting instance: %s, err: %s", rs.Primary.ID, err)
		}

		if v.InstanceID != rs.Primary.ID {
			return fmt.Errorf("The Dcs instance not found.")
		}
		instance = *v
		return nil
	}
}

func testAccDcsV1Instance_depr(instanceName string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_networking_secgroup_v2" "secgroup_1" {
  name        = "secgroup_1"
  description = "secgroup_1"
}
data "opentelekomcloud_dcs_az_v1" "az_1" {
  port = "8002"
  code = "%s"
}
data "opentelekomcloud_dcs_product_v1" "product_1" {
  spec_code = "dcs.master_standby"
}
resource "opentelekomcloud_dcs_instance_v1" "instance_1" {
  name              = "%s"
  engine_version    = "3.0"
  password          = "Hungarian_rapsody"
  engine            = "Redis"
  capacity          = 2
  vpc_id            = "%s"
  security_group_id = opentelekomcloud_networking_secgroup_v2.secgroup_1.id
  subnet_id         = "%s"
  available_zones = [data.opentelekomcloud_dcs_az_v1.az_1.id]
  product_id  = data.opentelekomcloud_dcs_product_v1.product_1.id
  save_days   = 1
  backup_type = "manual"
  begin_at    = "00:00-01:00"
  period_type = "weekly"
  backup_at = [4]
  depends_on = [
    "data.opentelekomcloud_dcs_product_v1.product_1",
  "opentelekomcloud_networking_secgroup_v2.secgroup_1"]
}
	`, env.OS_AVAILABILITY_ZONE, instanceName, env.OS_VPC_ID, env.OS_NETWORK_ID)
}

func testAccDcsV1Instance_basic(instanceName string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_networking_secgroup_v2" "secgroup_1" {
  name        = "secgroup_1"
  description = "secgroup_1"
}
data "opentelekomcloud_dcs_az_v1" "az_1" {
  port = "8002"
  code = "%s"
}
data "opentelekomcloud_dcs_product_v1" "product_1" {
  spec_code = "dcs.master_standby"
}
resource "opentelekomcloud_dcs_instance_v1" "instance_1" {
  name              = "%s"
  engine_version    = "3.0"
  password          = "Hungarian_rapsody"
  engine            = "Redis"
  capacity          = 2
  vpc_id            = "%s"
  security_group_id = opentelekomcloud_networking_secgroup_v2.secgroup_1.id
  subnet_id         = "%s"
  available_zones = [data.opentelekomcloud_dcs_az_v1.az_1.id]
  product_id  = data.opentelekomcloud_dcs_product_v1.product_1.id
  backup_policy {
    backup_type = "manual"
    begin_at    = "00:00-01:00"
    period_type = "weekly"
    backup_at = [4]
    save_days = 1
  }
  depends_on = [
    "data.opentelekomcloud_dcs_product_v1.product_1",
    "opentelekomcloud_networking_secgroup_v2.secgroup_1"]
}
	`, env.OS_AVAILABILITY_ZONE, instanceName, env.OS_VPC_ID, env.OS_NETWORK_ID)
}
func testAccDcsV1Instance_updated(instanceName string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_networking_secgroup_v2" "secgroup_1" {
  name        = "secgroup_1"
  description = "secgroup_1"
}
data "opentelekomcloud_dcs_az_v1" "az_1" {
  port = "8002"
  code = "%s"
}
data "opentelekomcloud_dcs_product_v1" "product_1" {
  spec_code = "dcs.master_standby"
}
resource "opentelekomcloud_dcs_instance_v1" "instance_1" {
  name              = "%s"
  engine_version    = "3.0"
  password          = "Hungarian_rapsody"
  engine            = "Redis"
  capacity          = 2
  vpc_id            = "%s"
  security_group_id = opentelekomcloud_networking_secgroup_v2.secgroup_1.id
  subnet_id         = "%s"
  available_zones = [data.opentelekomcloud_dcs_az_v1.az_1.id]
  product_id  = data.opentelekomcloud_dcs_product_v1.product_1.id
  backup_policy {
    backup_type = "manual"
    begin_at    = "01:00-02:00"
    period_type = "weekly"
    backup_at = [1, 2, 4]
    save_days = 2
  }
  depends_on = [
    "data.opentelekomcloud_dcs_product_v1.product_1",
    "opentelekomcloud_networking_secgroup_v2.secgroup_1"]
}
	`, env.OS_AVAILABILITY_ZONE, instanceName, env.OS_VPC_ID, env.OS_NETWORK_ID)
}

func testAccDcsV1Instance_single(instanceName string) string {
	return fmt.Sprintf(`
resource "opentelekomcloud_networking_secgroup_v2" "secgroup_1" {
  name        = "secgroup_1"
  description = "secgroup_1"
}
data "opentelekomcloud_dcs_az_v1" "az_1" {
  port = "8002"
  code = "%s"
}
data "opentelekomcloud_dcs_product_v1" "product_1" {
  spec_code = "dcs.single_node"
}
resource "opentelekomcloud_dcs_instance_v1" "instance_1" {
  name              = "%s"
  engine_version    = "3.0.7"
  password          = "Hungarian_rapsody"
  engine            = "Redis"
  capacity          = 2
  vpc_id            = "%s"
  security_group_id = opentelekomcloud_networking_secgroup_v2.secgroup_1.id
  subnet_id         = "%s"
  available_zones = [data.opentelekomcloud_dcs_az_v1.az_1.id]
  product_id  = data.opentelekomcloud_dcs_product_v1.product_1.id
  depends_on = [
    "data.opentelekomcloud_dcs_product_v1.product_1",
    "opentelekomcloud_networking_secgroup_v2.secgroup_1"]
}
	`, env.OS_AVAILABILITY_ZONE, instanceName, env.OS_VPC_ID, env.OS_NETWORK_ID)
}
