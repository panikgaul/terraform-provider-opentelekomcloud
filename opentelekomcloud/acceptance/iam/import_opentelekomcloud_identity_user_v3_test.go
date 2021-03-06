package acceptance

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
)

func TestAccIdentityV3User_importBasic(t *testing.T) {
	resourceName := "opentelekomcloud_identity_user_v3.user_1"
	var userName = fmt.Sprintf("ACCPTTEST-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			common.TestAccPreCheck(t)
			common.TestAccPreCheckAdminOnly(t)
		},
		Providers:    common.TestAccProviders,
		CheckDestroy: testAccCheckIdentityV3UserDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIdentityV3User_basic(userName),
			},

			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				},
			},
		},
	})
}
