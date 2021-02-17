package kms

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/kms/v1/keys"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

func DataSourceKmsDataKeyV1() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKmsDataKeyV1Read,

		Schema: map[string]*schema.Schema{
			"key_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"encryption_context": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"datakey_length": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"plain_text": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cipher_text": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceKmsDataKeyV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)

	KmsDataKeyV1Client, err := config.KmsKeyV1Client(config.GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud kms key client: %s", err)
	}

	req := &keys.DataEncryptOpts{
		KeyID:             d.Get("key_id").(string),
		EncryptionContext: d.Get("encryption_context").(string),
		DatakeyLength:     d.Get("datakey_length").(string),
	}
	log.Printf("[DEBUG] KMS get data key for key: %s", d.Get("key_id").(string))
	v, err := keys.DataEncryptGet(KmsDataKeyV1Client, req).ExtractDataKey()
	if err != nil {
		return err
	}

	d.SetId(time.Now().UTC().String())
	d.Set("plain_text", v.PlainText)
	d.Set("cipher_text", v.CipherText)

	return nil
}