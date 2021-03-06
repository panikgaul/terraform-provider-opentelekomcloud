package vpc

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/opentelekomcloud/gophertelekomcloud"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/ports"

	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

func ResourceNetworkingPortV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkingPortV2Create,
		Read:   resourceNetworkingPortV2Read,
		Update: resourceNetworkingPortV2Update,
		Delete: resourceNetworkingPortV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"network_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"admin_state_up": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
			"mac_address": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"device_owner": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"no_security_groups": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
			},
			"device_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"fixed_ip": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"allowed_address_pairs": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Computed: true,
				Set:      allowedAddressPairsHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip_address": {
							Type:     schema.TypeString,
							Required: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"value_specs": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
			"all_fixed_ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceNetworkingPortV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)
	networkingClient, err := config.NetworkingV2Client(config.GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud networking client: %s", err)
	}

	asu, id := ExtractValFromNid(d.Get("network_id").(string))
	pAsu := resourcePortAdminStateUpV2(d)
	if !asu {
		pAsu = &asu
	}

	var securityGroups []string
	securityGroups = ResourcePortSecurityGroupsV2(d)
	noSecurityGroups := d.Get("no_security_groups").(bool)

	// Check and make sure an invalid security group configuration wasn't given.
	if noSecurityGroups && len(securityGroups) > 0 {
		return fmt.Errorf("Cannot have both no_security_groups and security_group_ids set")
	}

	createOpts := PortCreateOpts{
		ports.CreateOpts{
			Name:                d.Get("name").(string),
			AdminStateUp:        pAsu,
			NetworkID:           id,
			MACAddress:          d.Get("mac_address").(string),
			TenantID:            d.Get("tenant_id").(string),
			DeviceOwner:         d.Get("device_owner").(string),
			DeviceID:            d.Get("device_id").(string),
			FixedIPs:            resourcePortFixedIpsV2(d),
			AllowedAddressPairs: resourceAllowedAddressPairsV2(d),
		},
		common.MapValueSpecs(d),
	}

	if noSecurityGroups {
		securityGroups = []string{}
		createOpts.SecurityGroups = &securityGroups
	}

	// Only set SecurityGroups if one was specified.
	// Otherwise this would mimic the no_security_groups action.
	if len(securityGroups) > 0 {
		createOpts.SecurityGroups = &securityGroups
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	p, err := ports.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud Neutron network: %s", err)
	}
	log.Printf("[INFO] Network ID: %s", p.ID)

	log.Printf("[DEBUG] Waiting for OpenTelekomCloud Neutron Port (%s) to become available.", p.ID)

	stateConf := &resource.StateChangeConf{
		Target:     []string{"ACTIVE"},
		Refresh:    waitForNetworkPortActive(networkingClient, p.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	d.SetId(p.ID)

	return resourceNetworkingPortV2Read(d, meta)
}

func resourceNetworkingPortV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)
	networkingClient, err := config.NetworkingV2Client(config.GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud networking client: %s", err)
	}

	p, err := ports.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return common.CheckDeleted(d, err, "port")
	}

	log.Printf("[DEBUG] Retrieved Port %s: %+v", d.Id(), p)

	asu, _ := ExtractValSFromNid(d.Get("network_id").(string))
	nid := FormatNidFromValS(asu, p.NetworkID)
	d.Set("name", p.Name)
	d.Set("admin_state_up", p.AdminStateUp)
	d.Set("network_id", nid)
	d.Set("mac_address", p.MACAddress)
	d.Set("tenant_id", p.TenantID)
	d.Set("device_owner", p.DeviceOwner)
	if err := d.Set("security_group_ids", p.SecurityGroups); err != nil {
		return fmt.Errorf("[DEBUG] Error saving security_group_ids to state for OpenTelekomCloud port (%s): %s", d.Id(), err)
	}
	d.Set("device_id", p.DeviceID)

	// Create a slice of all returned Fixed IPs.
	// This will be in the order returned by the API,
	// which is usually alpha-numeric.
	var ips []string
	for _, ipObject := range p.FixedIPs {
		ips = append(ips, ipObject.IPAddress)
	}
	if err := d.Set("all_fixed_ips", ips); err != nil {
		return fmt.Errorf("[DEBUG] Error saving all_fixed_ips to state for OpenTelekomCloud port (%s): %s", d.Id(), err)
	}

	// Convert AllowedAddressPairs to list of map
	var pairs []map[string]interface{}
	for _, pairObject := range p.AllowedAddressPairs {
		pair := make(map[string]interface{})
		pair["ip_address"] = pairObject.IPAddress
		pair["mac_address"] = pairObject.MACAddress
		pairs = append(pairs, pair)
	}
	if err := d.Set("allowed_address_pairs", pairs); err != nil {
		return fmt.Errorf("[DEBUG] Error saving allowed_address_pairs to state for OpenTelekomCloud port (%s): %s", d.Id(), err)
	}

	d.Set("region", config.GetRegion(d))

	return nil
}

func resourceNetworkingPortV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)
	networkingClient, err := config.NetworkingV2Client(config.GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud networking client: %s", err)
	}

	noSecurityGroups := d.Get("no_security_groups").(bool)
	var hasChange bool

	// security_group_ids and allowed_address_pairs are able to send empty arrays
	// to denote the removal of each. But their default zero-value is translated
	// to "null", which has been reported to cause problems in vendor-modified
	// OpenTelekomCloud clouds. Therefore, we must set them in each request update.
	var updateOpts ports.UpdateOpts

	if d.HasChange("allowed_address_pairs") {
		hasChange = true
		aap := resourceAllowedAddressPairsV2(d)
		updateOpts.AllowedAddressPairs = &aap
	}

	if d.HasChange("no_security_groups") {
		if noSecurityGroups {
			hasChange = true
			v := []string{}
			updateOpts.SecurityGroups = &v
		}
	}

	if d.HasChange("security_group_ids") {
		hasChange = true
		securityGroups := ResourcePortSecurityGroupsV2(d)
		updateOpts.SecurityGroups = &securityGroups
	}

	if d.HasChange("name") {
		hasChange = true
		updateOpts.Name = d.Get("name").(string)
	}

	if d.HasChange("admin_state_up") {
		hasChange = true
		asu, _ := ExtractValFromNid(d.Get("network_id").(string))
		pAsu := resourcePortAdminStateUpV2(d)
		if !asu {
			pAsu = &asu
		}
		updateOpts.AdminStateUp = pAsu
	}

	if d.HasChange("device_owner") {
		hasChange = true
		updateOpts.DeviceOwner = d.Get("device_owner").(string)
	}

	if d.HasChange("device_id") {
		hasChange = true
		updateOpts.DeviceID = d.Get("device_id").(string)
	}

	if d.HasChange("fixed_ip") {
		hasChange = true
		updateOpts.FixedIPs = resourcePortFixedIpsV2(d)
	}

	if hasChange {
		log.Printf("[DEBUG] Updating Port %s with options: %+v", d.Id(), updateOpts)

		_, err = ports.Update(networkingClient, d.Id(), updateOpts).Extract()
		if err != nil {
			return fmt.Errorf("Error updating OpenTelekomCloud Neutron Network: %s", err)
		}
	}
	return resourceNetworkingPortV2Read(d, meta)
}

func resourceNetworkingPortV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*cfg.Config)
	networkingClient, err := config.NetworkingV2Client(config.GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenTelekomCloud networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForNetworkPortDelete(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenTelekomCloud Neutron Network: %s", err)
	}

	d.SetId("")
	return nil
}

func ResourcePortSecurityGroupsV2(d *schema.ResourceData) []string {
	rawSecurityGroups := d.Get("security_group_ids").(*schema.Set)
	groups := make([]string, rawSecurityGroups.Len())
	for i, raw := range rawSecurityGroups.List() {
		groups[i] = raw.(string)
	}
	return groups
}

func resourcePortFixedIpsV2(d *schema.ResourceData) interface{} {
	rawIP := d.Get("fixed_ip").([]interface{})

	if len(rawIP) == 0 {
		return nil
	}

	ip := make([]ports.IP, len(rawIP))
	for i, raw := range rawIP {
		rawMap := raw.(map[string]interface{})
		ip[i] = ports.IP{
			SubnetID:  rawMap["subnet_id"].(string),
			IPAddress: rawMap["ip_address"].(string),
		}
	}
	return ip
}

func resourceAllowedAddressPairsV2(d *schema.ResourceData) []ports.AddressPair {
	// ports.AddressPair
	rawPairs := d.Get("allowed_address_pairs").(*schema.Set).List()

	pairs := make([]ports.AddressPair, len(rawPairs))
	for i, raw := range rawPairs {
		rawMap := raw.(map[string]interface{})
		pairs[i] = ports.AddressPair{
			IPAddress:  rawMap["ip_address"].(string),
			MACAddress: rawMap["mac_address"].(string),
		}
	}
	return pairs
}

func resourcePortAdminStateUpV2(d *schema.ResourceData) *bool {
	value := false

	if raw, ok := d.GetOk("admin_state_up"); ok && raw == true {
		value = true
	}

	return &value
}

func allowedAddressPairsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s", m["ip_address"].(string)))

	return hashcode.String(buf.String())
}

func waitForNetworkPortActive(networkingClient *golangsdk.ServiceClient, portId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		p, err := ports.Get(networkingClient, portId).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenTelekomCloud Neutron Port: %+v", p)
		if p.Status == "DOWN" || p.Status == "ACTIVE" {
			return p, "ACTIVE", nil
		}

		return p, p.Status, nil
	}
}

func waitForNetworkPortDelete(networkingClient *golangsdk.ServiceClient, portId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenTelekomCloud Neutron Port %s", portId)

		p, err := ports.Get(networkingClient, portId).Extract()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenTelekomCloud Port %s", portId)
				return p, "DELETED", nil
			}
			return p, "ACTIVE", err
		}

		err = ports.Delete(networkingClient, portId).ExtractErr()
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenTelekomCloud Port %s", portId)
				return p, "DELETED", nil
			}
			return p, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenTelekomCloud Port %s still active.\n", portId)
		return p, "ACTIVE", nil
	}
}
