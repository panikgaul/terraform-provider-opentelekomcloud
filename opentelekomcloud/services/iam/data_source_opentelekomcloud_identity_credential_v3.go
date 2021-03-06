package iam

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/credentials"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

func DataSourceIdentityCredentialV3() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIdentityCredentialV3Read,
		Schema: map[string]*schema.Schema{
			"user_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"credentials": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"access": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"create_time": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceIdentityCredentialV3Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)
	client, err := config.IdentityV30Client()
	if err != nil {
		return fmt.Errorf("error creating identity v3.0 client: %s", err)
	}
	userID := d.Get("user_id").(string)
	credentialList, err := credentials.List(client, credentials.ListOpts{UserID: userID}).Extract()
	if err != nil {
		return fmt.Errorf("error retrieving AK/SK information: %s", err)
	}

	me := new(multierror.Error)
	for i, credential := range credentialList {
		credKey := fmt.Sprintf("credentials.%d.", i)
		me = multierror.Append(me,
			d.Set(credKey+"used_id", credential.UserID),
			d.Set(credKey+"access", credential.AccessKey),
			d.Set(credKey+"status", credential.Status),
			d.Set(credKey+"create_time", credential.CreateTime),
			d.Set(credKey+"description", credential.Description),
		)
	}
	return me.ErrorOrNil()
}
