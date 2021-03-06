package acceptance

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
)

func TestAccOTCRTSStackV1_importBasic(t *testing.T) {
	resourceName := "opentelekomcloud_rts_stack_v1.stack_1"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { common.TestAccPreCheck(t) },
		Providers:    common.TestAccProviders,
		CheckDestroy: testAccCheckOTCRTSStackV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRTSStackV1_basic,
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
