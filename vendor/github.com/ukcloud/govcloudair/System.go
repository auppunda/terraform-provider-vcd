package govcloudair

import (
	"net/url"
	"strings"
	"bytes"
	"encoding/xml"
	"fmt"
	types "github.com/ukcloud/govcloudair/types/v56"
)

type System struct {
	c *Client
}

func NewSystemClient(c *Client) *System {
	return &System{
		c: c,
	}
}

func (s *System) SystemAuthorize(user, pass string) error {
	u := s.c.HREF
	u.Path += "/sessions"
	req := s.c.NewRequest(map[string]string{}, "POST", u, nil)

	// Set Basic Authentication Header
	req.SetBasicAuth(user+"@System", pass)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/*+xml;version=5.5")

	resp, err := checkResp(s.c.Http.Do(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Store the authentication header
	s.c.VCDToken = resp.Header.Get("x-vcloud-authorization")
	s.c.VCDAuthHeader = "x-vcloud-authorization"

	return nil

}

//Creates an Organization based on settings, network, and org name
func CreateOrg(c *VCDClient, org string, fullOrgName string, isEnabled bool, canPublishCatalogs bool, vmQuota int) (Task, error) {

	settings := getOrgSettings(canPublishCatalogs, vmQuota)

	vcomp := &types.AdminOrg{
		Xmlns:       "http://www.vmware.com/vcloud/v1.5",
		Name:        org,
		IsEnabled:   isEnabled,
		FullName:    fullOrgName,
		OrgSettings: settings,
	}

	output, _ := xml.MarshalIndent(vcomp, "  ", "    ")

	u := c.Client.HREF
	u.Path += "/admin/orgs"

	b := bytes.NewBufferString(xml.Header + string(output))

	req := c.Client.NewRequest(map[string]string{}, "POST", u, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.organization+xml")

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error instantiating a new Org: %s", err)
	}

	task := NewTask(&c.Client)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding task response: %s", err)
	}

	return *task, nil

}


func GetVDCFromName(c *VCDClient, vdcname string, orgname string) (Vdc, error) {

	o, err := GetOrgFromName(c, orgname)

	if err != nil {
		return Vdc{}, fmt.Errorf("Could not find org: %v", err)
	}

	HREF := ""
	for _, a := range o.Org.Link {
		if a.Type == "application/vnd.vmware.vcloud.vdc+xml" && a.Name == vdcname {
			HREF = a.HREF
			break
		}
	}

	if HREF == "" {
		return Vdc{}, fmt.Errorf("Error finding VDC from VDCName")
	}

	u, err := url.ParseRequestURI(HREF)

	if err != nil {
		return Vdc{}, fmt.Errorf("Error retrieving VDC: %v", err)
	}
	req := c.Client.NewRequest(map[string]string{}, "GET", *u , nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Vdc{}, fmt.Errorf("error retreiving vdc: %s", err)
	}

	vdc := NewVdc(&c.Client)

	if err = decodeBody(resp, vdc.Vdc); err != nil {
		return Vdc{}, fmt.Errorf("error decoding vdc response: %s", err)
	}

	// The request was successful
	return *vdc, nil
}

func getOrgHREF(c *VCDClient, orgname string) (string, error) {
	s := c.Client.HREF
	s.Path += "/org"
	req := c.Client.NewRequest(map[string]string{}, "GET", s , nil)


	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return "", fmt.Errorf("error retreiving vdc: %s", err)
	}

	orgList := new(types.OrgList)
	if err = decodeBody(resp, orgList); err != nil {
		return "", fmt.Errorf("error decoding vdc response: %s", err)
	}

	for _, a := range orgList.Org  {
		if a.Name == orgname {
			return a.HREF, nil
		}
	}

	return "", fmt.Errorf("Couldn't find org with name: %s", orgname)


}

func GetOrgFromName(c *VCDClient, orgname string) (Org, error) {

	o := NewOrg(&c.Client)
	
	HREF, err := getOrgHREF(c, orgname)
	if err != nil {
		return Org{}, fmt.Errorf("Cannot find OrgHREF: %s", err)
	}

	u, err := url.ParseRequestURI(HREF)
	if err != nil {
		return Org{}, fmt.Errorf("Error parsing org href: %v", err)
	}

	req := c.Client.NewRequest(map[string]string{}, "GET", *u , nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Org{}, fmt.Errorf("error retreiving org: %s", err)
	}

	if err = decodeBody(resp, o.Org); err != nil {
		return Org{}, fmt.Errorf("error decoding org response: %s", err)
	}

	return *o, nil
}

func GetOrgFromAdminOrg(c *VCDClient, adminOrg AdminOrg) (Org, error) {
	o := NewOrg(&c.Client)

	s := c.Client.HREF
	s.Path += "/org/" + adminOrg.AdminOrg.ID[15:]

	req := c.Client.NewRequest(map[string]string{}, "GET", s , nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Org{}, fmt.Errorf("error fetching Org : %s", err) 
	}

	if err = decodeBody(resp, o.Org); err != nil {
		return Org{}, fmt.Errorf("error decoding org response: %s", err)
	}

	return *o, nil
}

func GetAdminOrgFromName(c *VCDClient, orgname string) (AdminOrg, error) {
	o := NewAdminOrg(&c.Client)
	
	HREF, err := getOrgHREF(c, orgname)
	if err != nil {
		return AdminOrg{}, fmt.Errorf("Cannot find OrgHREF: %s", err)
	}

	u := c.Client.HREF
	u.Path += "/admin/org/" + strings.Split(HREF, "/org/")[1]

	req := c.Client.NewRequest(map[string]string{}, "GET", u , nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return AdminOrg{}, fmt.Errorf("error retreiving org: %s", err)
	}

	if err = decodeBody(resp, o.AdminOrg); err != nil {
		return AdminOrg{}, fmt.Errorf("error decoding org response: %s", err)
	}

	return *o, nil
}

func GetAdminOrgFromOrg(c *VCDClient, org Org) (AdminOrg, error) {
	o := NewAdminOrg(&c.Client)

	s := c.Client.HREF
	s.Path += "/admin/org/" +org.Org.ID[15:]

	req := c.Client.NewRequest(map[string]string{}, "GET", s , nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return AdminOrg{}, fmt.Errorf("error fetching Org : %s", err) 
	}

	if err = decodeBody(resp, o.AdminOrg); err != nil {
		return AdminOrg{}, fmt.Errorf("error decoding org response: %s", err)
	}

	return *o, nil
}

//Fetches an org using the Org ID, which is the UUID in the Org HREF.
func GetAdminOrgById(c *VCDClient, orgId string) (AdminOrg, error) {
	u := c.Client.HREF
	u.Path += "/admin/org/" + orgId

	req := c.Client.NewRequest(map[string]string{}, "GET", u, nil)

	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.org+xml")

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return AdminOrg{}, fmt.Errorf("error getting Org %s: %s", orgId, err)
	}

	org := NewAdminOrg(&c.Client)

	if err = decodeBody(resp, org.AdminOrg); err != nil {
		return AdminOrg{}, fmt.Errorf("error decoding org response: %s", err)
	}

	return *org, nil
}

func getOrgSettings(canPublishCatalogs bool, vmQuota int) *types.OrgSettings {
	var settings *types.OrgSettings
	if vmQuota != -1 {
		settings = &types.OrgSettings{
			General: &types.OrgGeneralSettings{
				CanPublishCatalogs: canPublishCatalogs,
				DeployedVMQuota:    vmQuota,
				StoredVMQuota:      vmQuota,
			},
		}
	} else {
		settings = &types.OrgSettings{
			General: &types.OrgGeneralSettings{
				CanPublishCatalogs: canPublishCatalogs,
			},
		}
	}
	return settings
}
