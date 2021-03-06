package lts

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/lts/v2/loggroups"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

func ResourceLTSGroupV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceGroupV2Create,
		Read:   resourceGroupV2Read,
		Delete: resourceGroupV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ttl_in_days": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceGroupV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)
	client, err := config.LtsV2Client(config.GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud LTS client: %s", err)
	}

	createOpts := &loggroups.CreateOpts{
		LogGroupName: d.Get("group_name").(string),
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)

	groupCreate, err := loggroups.Create(client, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating log group: %s", err)
	}

	d.SetId(groupCreate.ID)
	return resourceGroupV2Read(d, meta)
}

func resourceGroupV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)
	client, err := config.LtsV2Client(config.GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud LTS client: %s", err)
	}

	group, err := loggroups.Get(client, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error getting OpenTelekomCloud log group %s: %s", d.Id(), err)
	}

	log.Printf("[DEBUG] Retrieved Cluster %s: %#v", d.Id(), group)
	d.SetId(group.ID)
	d.Set("group_name", group.Name)
	d.Set("ttl_in_days", group.TTLinDays)
	return nil
}

func resourceGroupV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)
	client, err := config.LtsV2Client(config.GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud LTS client: %s", err)
	}

	err = loggroups.Delete(client, d.Id()).ExtractErr()
	if err != nil {
		return common.CheckDeleted(d, err, "Error deleting log group")
	}

	d.SetId("")
	return nil
}
