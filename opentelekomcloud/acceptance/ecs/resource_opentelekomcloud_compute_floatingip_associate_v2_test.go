package acceptance

import (
	"fmt"
	"testing"

	"github.com/opentelekomcloud/gophertelekomcloud"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/compute/v2/servers"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/layer3/floatingips"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/env"
	vpc "github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/vpc"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/services/ecs"
)

func TestAccComputeV2FloatingIPAssociate_basic(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { common.TestAccPreCheck(t) },
		Providers:    common.TestAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPAssociateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeV2FloatingIPAssociate_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("opentelekomcloud_compute_instance_v2.instance_1", &instance),
					vpc.TestAccCheckNetworkingV2FloatingIPExists("opentelekomcloud_networking_floatingip_v2.fip_1", &fip),
					testAccCheckComputeV2FloatingIPAssociateAssociated(&fip, &instance, 1),
				),
			},
		},
	})
}

func TestAccComputeV2FloatingIPAssociate_fixedIP(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { common.TestAccPreCheck(t) },
		Providers:    common.TestAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPAssociateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeV2FloatingIPAssociate_fixedIP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("opentelekomcloud_compute_instance_v2.instance_1", &instance),
					vpc.TestAccCheckNetworkingV2FloatingIPExists("opentelekomcloud_networking_floatingip_v2.fip_1", &fip),
					testAccCheckComputeV2FloatingIPAssociateAssociated(&fip, &instance, 1),
				),
			},
		},
	})
}

func TestAccComputeV2FloatingIPAssociate_attachToFirstNetwork(t *testing.T) {
	var instance servers.Server
	var fip floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { common.TestAccPreCheck(t) },
		Providers:    common.TestAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPAssociateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeV2FloatingIPAssociate_attachToFirstNetwork,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("opentelekomcloud_compute_instance_v2.instance_1", &instance),
					vpc.TestAccCheckNetworkingV2FloatingIPExists("opentelekomcloud_networking_floatingip_v2.fip_1", &fip),
					testAccCheckComputeV2FloatingIPAssociateAssociated(&fip, &instance, 1),
				),
			},
		},
	})
}

func TestAccComputeV2FloatingIPAssociate_attachNew(t *testing.T) {
	var instance servers.Server
	var fip_1 floatingips.FloatingIP
	var fip_2 floatingips.FloatingIP

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { common.TestAccPreCheck(t) },
		Providers:    common.TestAccProviders,
		CheckDestroy: testAccCheckComputeV2FloatingIPAssociateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeV2FloatingIPAssociate_attachNew_1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("opentelekomcloud_compute_instance_v2.instance_1", &instance),
					vpc.TestAccCheckNetworkingV2FloatingIPExists("opentelekomcloud_networking_floatingip_v2.fip_1", &fip_1),
					vpc.TestAccCheckNetworkingV2FloatingIPExists("opentelekomcloud_networking_floatingip_v2.fip_2", &fip_2),
					testAccCheckComputeV2FloatingIPAssociateAssociated(&fip_1, &instance, 1),
				),
			},
			{
				Config: testAccComputeV2FloatingIPAssociate_attachNew_2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeV2InstanceExists("opentelekomcloud_compute_instance_v2.instance_1", &instance),
					vpc.TestAccCheckNetworkingV2FloatingIPExists("opentelekomcloud_networking_floatingip_v2.fip_1", &fip_1),
					vpc.TestAccCheckNetworkingV2FloatingIPExists("opentelekomcloud_networking_floatingip_v2.fip_2", &fip_2),
					testAccCheckComputeV2FloatingIPAssociateAssociated(&fip_2, &instance, 1),
				),
			},
		},
	})
}

func testAccCheckComputeV2FloatingIPAssociateDestroy(s *terraform.State) error {
	config := common.TestAccProvider.Meta().(*cfg.Config)
	computeClient, err := config.ComputeV2Client(env.OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud compute client: %s", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opentelekomcloud_compute_floatingip_associate_v2" {
			continue
		}

		floatingIP, instanceId, _, err := ecs.ParseComputeFloatingIPAssociateId(rs.Primary.ID)
		if err != nil {
			return err
		}

		instance, err := servers.Get(computeClient, instanceId).Extract()
		if err != nil {
			// If the error is a 404, then the instance does not exist,
			// and therefore the floating IP cannot be associated to it.
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				return nil
			}
			return err
		}

		// But if the instance still exists, then walk through its known addresses
		// and see if there's a floating IP.
		for _, networkAddresses := range instance.Addresses {
			for _, element := range networkAddresses.([]interface{}) {
				address := element.(map[string]interface{})
				if address["OS-EXT-IPS:type"] == "floating" || address["OS-EXT-IPS:type"] == "fixed" {
					return fmt.Errorf("Floating IP %s is still attached to instance %s", floatingIP, instanceId)
				}
			}
		}
	}

	return nil
}

func testAccCheckComputeV2FloatingIPAssociateAssociated(
	fip *floatingips.FloatingIP, instance *servers.Server, n int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := common.TestAccProvider.Meta().(*cfg.Config)
		computeClient, err := config.ComputeV2Client(env.OS_REGION_NAME)

		newInstance, err := servers.Get(computeClient, instance.ID).Extract()
		if err != nil {
			return err
		}

		// Walk through the instance's addresses and find the match
		i := 0
		for _, networkAddresses := range newInstance.Addresses {
			i += 1
			if i != n {
				continue
			}
			for _, element := range networkAddresses.([]interface{}) {
				address := element.(map[string]interface{})
				if (address["OS-EXT-IPS:type"] == "floating" && address["addr"] == fip.FloatingIP) ||
					(address["OS-EXT-IPS:type"] == "fixed" && address["addr"] == fip.FixedIP) {
					return nil
				}
			}
		}
		return fmt.Errorf("Floating IP %s was not attached to instance %s", fip.FloatingIP, instance.ID)
	}
}

var testAccComputeV2FloatingIPAssociate_basic = fmt.Sprintf(`
resource "opentelekomcloud_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  network {
    uuid = "%s"
  }
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_1" {
}

resource "opentelekomcloud_compute_floatingip_associate_v2" "fip_1" {
  floating_ip = opentelekomcloud_networking_floatingip_v2.fip_1.address
  instance_id = opentelekomcloud_compute_instance_v2.instance_1.id
}
`, env.OS_NETWORK_ID)

var testAccComputeV2FloatingIPAssociate_fixedIP = fmt.Sprintf(`
resource "opentelekomcloud_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  network {
    uuid = "%s"
  }
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_1" {
}

resource "opentelekomcloud_compute_floatingip_associate_v2" "fip_1" {
  floating_ip = opentelekomcloud_networking_floatingip_v2.fip_1.address
  instance_id = opentelekomcloud_compute_instance_v2.instance_1.id
  fixed_ip = opentelekomcloud_compute_instance_v2.instance_1.access_ip_v4
}
`, env.OS_NETWORK_ID)

var testAccComputeV2FloatingIPAssociate_attachToFirstNetwork = fmt.Sprintf(`
resource "opentelekomcloud_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  network {
    uuid = "%s"
  }
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_1" {
}

resource "opentelekomcloud_compute_floatingip_associate_v2" "fip_1" {
  floating_ip = opentelekomcloud_networking_floatingip_v2.fip_1.address
  instance_id = opentelekomcloud_compute_instance_v2.instance_1.id
  fixed_ip = opentelekomcloud_compute_instance_v2.instance_1.network.0.fixed_ip_v4
}
`, env.OS_NETWORK_ID)

var testAccComputeV2FloatingIPAssociate_attachToSecondNetwork = fmt.Sprintf(`
resource "opentelekomcloud_networking_network_v2" "network_1" {
  name = "network_1"
}

resource "opentelekomcloud_networking_subnet_v2" "subnet_1" {
  name = "subnet_1"
  network_id = opentelekomcloud_networking_network_v2.network_1.id
  cidr = "192.168.1.0/24"
  ip_version = 4
  enable_dhcp = true
  no_gateway = true
}

resource "opentelekomcloud_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  network {
    uuid = opentelekomcloud_networking_network_v2.network_1.id
  }

  network {
    uuid = "%s"
  }
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_1" {
}

resource "opentelekomcloud_compute_floatingip_associate_v2" "fip_1" {
  floating_ip = opentelekomcloud_networking_floatingip_v2.fip_1.address
  instance_id = opentelekomcloud_compute_instance_v2.instance_1.id
  fixed_ip = opentelekomcloud_compute_instance_v2.instance_1.network.1.fixed_ip_v4
}
`, env.OS_NETWORK_ID)

var testAccComputeV2FloatingIPAssociate_attachNew_1 = fmt.Sprintf(`
resource "opentelekomcloud_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  network {
    uuid = "%s"
  }
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_1" {
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_2" {
}

resource "opentelekomcloud_compute_floatingip_associate_v2" "fip_1" {
  floating_ip = opentelekomcloud_networking_floatingip_v2.fip_1.address
  instance_id = opentelekomcloud_compute_instance_v2.instance_1.id
}
`, env.OS_NETWORK_ID)

var testAccComputeV2FloatingIPAssociate_attachNew_2 = fmt.Sprintf(`
resource "opentelekomcloud_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]
  network {
    uuid = "%s"
  }
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_1" {
}

resource "opentelekomcloud_networking_floatingip_v2" "fip_2" {
}

resource "opentelekomcloud_compute_floatingip_associate_v2" "fip_1" {
  floating_ip = opentelekomcloud_networking_floatingip_v2.fip_2.address
  instance_id = opentelekomcloud_compute_instance_v2.instance_1.id
}
`, env.OS_NETWORK_ID)
