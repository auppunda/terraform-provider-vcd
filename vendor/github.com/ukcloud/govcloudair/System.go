package govcloudair

import (
	//"net/url"
	//"sync"
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
func (s *System) CreateOrg(org string, fullOrgName string, isEnabled bool, canPublishCatalogs bool, vmQuota int) (Task, error) {

	settings := getOrgSettings(canPublishCatalogs, vmQuota)

	vcomp := &types.AdminOrg{
		Xmlns:       "http://www.vmware.com/vcloud/v1.5",
		Name:        org,
		IsEnabled:   isEnabled,
		FullName:    fullOrgName,
		OrgSettings: settings,
	}

	output, _ := xml.MarshalIndent(vcomp, "  ", "    ")

	u := s.c.HREF
	u.Path += "/admin/orgs"

	b := bytes.NewBufferString(xml.Header + string(output))

	req := s.c.NewRequest(map[string]string{}, "POST", u, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.organization+xml")

	resp, err := checkResp(s.c.Http.Do(req))
	if err != nil {
		return Task{}, fmt.Errorf("error instantiating a new Org: %s", err)
	}

	task := NewTask(s.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding task response: %s", err)
	}

	return *task, nil

}

//Fetches an org using the Org ID, which is the UUID in the Org HREF.
func (s *System) GetAdminOrgById(orgId string) (AdminOrg, error) {
	u := s.c.HREF
	u.Path += "/admin/org/" + orgId

	req := s.c.NewRequest(map[string]string{}, "GET", u, nil)

	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.org+xml")

	resp, err := checkResp(s.c.Http.Do(req))
	if err != nil {
		return AdminOrg{}, fmt.Errorf("error getting Org %s: %s", orgId, err)
	}

	org := NewAdminOrg(s.c)

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
