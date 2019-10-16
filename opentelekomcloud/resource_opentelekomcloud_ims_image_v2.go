package opentelekomcloud

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/huaweicloud/golangsdk"
	imageservice_v2 "github.com/huaweicloud/golangsdk/openstack/imageservice/v2/images"
	"github.com/huaweicloud/golangsdk/openstack/ims/v2/cloudimages"
)

func resourceImsImageV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceImsImageV2Create,
		Read:   resourceImsImageV2Read,
		Update: resourceImsImageV2Update,
		Delete: resourceImagesImageV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"image_tags": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},
			"max_ram": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"min_ram": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},
			// instance_id is required for creating an image from an ECS
			"instance_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"image_url"},
			},
			// image_url and min_disk are required for creating an image from an OBS
			"image_url": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"instance_id"},
			},
			"min_disk": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      false,
				ConflictsWith: []string{"instance_id"},
			},
			// following are valid for creating an image from an OBS
			"os_version": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"is_config": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"cmk_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"ECS", "FusionCompute", "BMS", "Ironic",
				}, true),
			},
			// following are additional attributus
			"visibility": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"data_origin": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"disk_format": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"image_size": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceContainerImageTags(d *schema.ResourceData) []cloudimages.ImageTag {
	var tags []cloudimages.ImageTag

	image_tags := d.Get("image_tags").(map[string]interface{})
	for key, val := range image_tags {
		tagRequest := cloudimages.ImageTag{
			Key:   key,
			Value: val.(string),
		}
		tags = append(tags, tagRequest)
	}
	return tags
}

func getImagetagsToList(d *schema.ResourceData) []string {
	var tags []string

	image_tags := d.Get("image_tags").(map[string]interface{})
	for key, val := range image_tags {
		// key.value
		tagRequest := key + "." + val.(string)
		tags = append(tags, tagRequest)
	}
	return tags
}

func resourceImsImageV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ims_Client, err := config.imageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud image client: %s", err)
	}

	if !hasFilledOpt(d, "instance_id") && !hasFilledOpt(d, "image_url") {
		return fmt.Errorf("Error creating OpenTelekomCloud IMS: " +
			"Either 'instance_id' or 'image_url' must be specified")
	}

	v := new(cloudimages.JobResponse)
	image_tags := resourceContainerImageTags(d)
	if hasFilledOpt(d, "instance_id") {
		createOpts := &cloudimages.CreateByServerOpts{
			Name:        d.Get("name").(string),
			Description: d.Get("description").(string),
			InstanceId:  d.Get("instance_id").(string),
			MaxRam:      d.Get("max_ram").(int),
			MinRam:      d.Get("min_ram").(int),
			ImageTags:   image_tags,
		}
		log.Printf("[DEBUG] Create Options: %#v", createOpts)
		v, err = cloudimages.CreateImageByServer(ims_Client, createOpts).ExtractJobResponse()
	} else {
		if !hasFilledOpt(d, "min_disk") {
			return fmt.Errorf("Error creating OpenTelekomCloud IMS: 'min_disk' must be specified")
		}

		createOpts := &cloudimages.CreateByOBSOpts{
			Name:        d.Get("name").(string),
			Description: d.Get("description").(string),
			ImageUrl:    d.Get("image_url").(string),
			MinDisk:     d.Get("min_disk").(int),
			MaxRam:      d.Get("max_ram").(int),
			MinRam:      d.Get("min_ram").(int),
			OsVersion:   d.Get("os_version").(string),
			IsConfig:    d.Get("is_config").(bool),
			CmkId:       d.Get("cmk_id").(string),
			Type:        d.Get("type").(string),
			ImageTags:   image_tags,
		}
		log.Printf("[DEBUG] Create Options: %#v", createOpts)
		v, err = cloudimages.CreateImageByOBS(ims_Client, createOpts).ExtractJobResponse()
	}

	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud IMS: %s", err)
	}
	log.Printf("[INFO] IMS Job ID: %s", v.JobID)

	// Wait for the ims to become available.
	log.Printf("[DEBUG] Waiting for IMS to become available")
	err = cloudimages.WaitForJobSuccess(ims_Client, int(d.Timeout(schema.TimeoutCreate)/time.Second), v.JobID)
	if err != nil {
		return err
	}

	entity, err := cloudimages.GetJobEntity(ims_Client, v.JobID, "image_id")
	if err != nil {
		return err
	}

	if id, ok := entity.(string); ok {
		log.Printf("[INFO] IMS ID: %s", id)
		// Store the ID now
		d.SetId(id)
		return resourceImsImageV2Read(d, meta)
	}
	return fmt.Errorf("Unexpected conversion error in resourceImsImageV2Create.")
}

func getCloudimage(client *golangsdk.ServiceClient, id string) (*cloudimages.Image, error) {
	listOpts := &cloudimages.ListOpts{
		ID:    id,
		Limit: 1,
	}
	allPages, err := cloudimages.List(client, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("Unable to query images: %s", err)
	}

	allImages, err := cloudimages.ExtractImages(allPages)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve images: %s", err)
	}

	if len(allImages) < 1 {
		return nil, fmt.Errorf("Unable to find images %s: Maybe not existed", id)
	}

	img := allImages[0]
	if img.ID != id {
		return nil, fmt.Errorf("Unexpected images ID")
	}
	log.Printf("[DEBUG] Retrieved Image %s: %#v", id, img)
	return &img, nil
}

func resourceImsImageV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ims_Client, err := config.imageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud image client: %s", err)
	}

	img, err := getCloudimage(ims_Client, d.Id())
	if err != nil {
		return fmt.Errorf("Image %s not found: %s", d.Id(), err)
	}
	log.Printf("[DEBUG] Retrieved Image %s: %#v", d.Id(), img)

	d.Set("name", img.Name)
	d.Set("visibility", img.Visibility)
	d.Set("file", img.File)
	d.Set("schema", img.Schema)
	d.Set("data_origin", img.DataOrigin)
	d.Set("disk_format", img.DiskFormat)
	d.Set("image_size", img.ImageSize)
	return nil
}

func resourceImsImageV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	ims_Client, err := config.imageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud image client: %s", err)
	}

	updateOpts := make(imageservice_v2.UpdateOpts, 0)

	if d.HasChange("name") {
		v := imageservice_v2.ReplaceImageName{NewName: d.Get("name").(string)}
		updateOpts = append(updateOpts, v)
	}

	if d.HasChange("image_tags") {
		updata_tags := getImagetagsToList(d)
		v := imageservice_v2.ReplaceImageTags{
			NewTags: updata_tags,
		}
		updateOpts = append(updateOpts, v)
	}

	log.Printf("[DEBUG] Update Options: %#v", updateOpts)

	_, err = imageservice_v2.Update(ims_Client, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating image: %s", err)
	}

	return resourceImsImageV2Read(d, meta)
}
