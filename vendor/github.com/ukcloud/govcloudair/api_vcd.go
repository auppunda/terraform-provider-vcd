package govcloudair

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
	"encoding/xml"
	"bytes"
	"strings"

	types "github.com/ukcloud/govcloudair/types/v56"
)

type VCDClient struct {
	OrgHREF     url.URL // vCloud Director OrgRef
	Org         Org     // Org
	OrgVdc      Vdc     // Org vDC
	Client      Client  // Client for the underlying VCD instance
	sessionHREF url.URL // HREF for the session API
	QueryHREF   url.URL // HREF for the query API
	HREF 		url.URL // normal API endpoint
	Mutex       sync.Mutex
}

type supportedVersions struct {
	VersionInfo struct {
		Version  string `xml:"Version"`
		LoginUrl string `xml:"LoginUrl"`
	} `xml:"VersionInfo"`
}

func (c *VCDClient) vcdloginurl() error {

	s := c.Client.VCDVDCHREF
	s.Path += "/versions"

	// No point in checking for errors here
	req := c.Client.NewRequest(map[string]string{}, "GET", s, nil)

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	supportedVersions := new(supportedVersions)

	err = decodeBody(resp, supportedVersions)

	if err != nil {
		return fmt.Errorf("error decoding versions response: %s", err)
	}

	u, err := url.Parse(supportedVersions.VersionInfo.LoginUrl)
	if err != nil {
		return fmt.Errorf("couldn't find a LoginUrl in versions")
	}
	c.sessionHREF = *u
	return nil
}

func (c *VCDClient) vcdauthorize(user, pass, org string) error {

	if user == "" {
		user = os.Getenv("VCLOUD_USERNAME")
	}

	if pass == "" {
		pass = os.Getenv("VCLOUD_PASSWORD")
	}

	if org == "" {
		org = os.Getenv("VCLOUD_ORG")
	}

	/* GETTING TOKEN FOR ADMINISTRATIVE PURPOSES */
	// No point in checking for errors here
	req := c.Client.NewRequest(map[string]string{}, "POST", c.sessionHREF, nil)

	// Set Basic Authentication Header
	req.SetBasicAuth(user+"@System", pass)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/*+xml;version=5.5")

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()


	// Store the authentication header
	c.Client.VCDToken = resp.Header.Get("x-vcloud-authorization")
	c.Client.VCDAuthHeader = "x-vcloud-authorization"

	//gets HREF for normal api purposes
	s := c.sessionHREF
	sArr := strings.Split(s.Path, "/session")
	s.Path = sArr[0]
	c.HREF = s

	/* GETTING SPECIFIC ORG AND VDC SO THAT IT DOESN'T BREAK EXISTING CODE, MAKES ANOTHER REQUEST TO THE ORG SO THAT IT CAN STORE
	ORG HREF and VDCHREF. THIS MIGHT BE REMOVED IN THE FUTURE IF WE DON'T HAVE THE USER CONNECT TO A SPECIFIC ORG BUT FOR RIGHT NOW
	IT NEEDS TO BE THERE. IF WE DONT DO THIS WE HAVE TO CHANGE THE API CODE FOR A LOT OF OTHER THINGS*/
	req = c.Client.NewRequest(map[string]string{}, "POST", c.sessionHREF, nil)

	// Set Basic Authentication Header
	req.SetBasicAuth(user+"@"+org, pass)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/*+xml;version=5.5")

	resp, err = checkResp(c.Client.Http.Do(req))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	session := new(session)
	err = decodeBody(resp, session)

	if err != nil {
		fmt.Errorf("error decoding session response: %s", err)
	}

	org_found := false
	// Loop in the session struct to find the organization and query api.
	for _, s := range session.Link {
		if s.Type == "application/vnd.vmware.vcloud.org+xml" && s.Rel == "down" {
			u, err := url.Parse(s.HREF)
			if err != nil {
				return fmt.Errorf("couldn't find a Organization in current session, %v", err)
			}
			c.OrgHREF = *u
			org_found = true
		}
		if s.Type == "application/vnd.vmware.vcloud.query.queryList+xml" && s.Rel == "down" {
			u, err := url.Parse(s.HREF)
			if err != nil {
				return fmt.Errorf("couldn't find a Query API in current session, %v", err)
			}
			c.QueryHREF = *u
		}
	}
	if !org_found {
		return fmt.Errorf("couldn't find a Organization in current session")
	}

	// Loop in the session struct to find the session url.
	session_found := false
	for _, s := range session.Link {
		if s.Rel == "remove" {
			u, err := url.Parse(s.HREF)
			if err != nil {
				return fmt.Errorf("couldn't find a logout HREF in current session, %v", err)
			}
			c.sessionHREF = *u
			session_found = true
		}
	}
	if !session_found {
		return fmt.Errorf("couldn't find a logout HREF in current session")
	}

	return nil
}

//update organization
func (c *VCDClient) UpdateOrg(vcomp *types.OrgParams, orgId string) (Task, error) {
	output, _ := xml.MarshalIndent(vcomp, "  ", "    ")

	s := c.HREF
	s.Path += "/admin/org/" + orgId

	b := bytes.NewBufferString(xml.Header + string(output))

	req := c.Client.NewRequest(map[string]string{}, "PUT", s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.organization+xml")

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Task{} , fmt.Errorf("error instantiating a new Org: %s", err)
	}

	task := NewTask(&c.Client)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding task response: %s", err)
	}

	return *task, nil
}

//gets an organization given an org_id
func (c *VCDClient) GetOrg(orgId string) (bool, Org, error) {
	s := c.HREF
	s.Path += "/org/" + orgId

	req := c.Client.NewRequest(map[string]string{}, "GET", s, nil)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.organization+xml")

	resp , err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return false, Org{}, fmt.Errorf("error getting Org %s: %s", orgId, err)
	}

	org := NewOrg(&c.Client)

	if err = decodeBody(resp, org.Org); err != nil {
		return false, Org{}, fmt.Errorf("error decoding org response: %s", err)
	}


	return true, *org, nil
}

//Creates an Organization based on settings, network, and org name
func (c *VCDClient) CreateOrg(org string, fullOrgName string, settings types.OrgSettings, isEnabled bool) (Task, string, error) {
	vcomp := &types.OrgParams{
		Xmlns:       "http://www.vmware.com/vcloud/v1.5",
		Name: org,
		IsEnabled: isEnabled,
		FullName: fullOrgName,
		OrgSettings: &settings,

	}


	output, _ := xml.MarshalIndent(vcomp, "  ", "    ")

	s := c.HREF
	s.Path += "/admin/orgs"

	b := bytes.NewBufferString(xml.Header + string(output))

	req := c.Client.NewRequest(map[string]string{}, "POST", s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.admin.organization+xml")

	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Task{}, "" , fmt.Errorf("error instantiating a new Org: %s", err)
	}

	task := NewTask(&c.Client)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, "", fmt.Errorf("error decoding task response: %s", err)
	}

	return *task, task.Task.ID , nil

}

//deletes an organization given an org_id
func (c *VCDClient) DeleteOrg(orgId string) (bool, error) {
	s := c.HREF
	s.Path += "/admin/org/" + orgId + "/action/disable"

	req := c.Client.NewRequest(map[string]string{}, "POST", s, nil)

	_ , err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return false, fmt.Errorf("error getting Org %s: %s", orgId, err)
	}

	s = c.HREF
	s.Path += "/admin/org/" + orgId

	req = c.Client.NewRequest(map[string]string{}, "DELETE", s, nil)

	_ , err = checkResp(c.Client.Http.Do(req))

	if err != nil {
		return false, fmt.Errorf("error getting Org %s: %s", orgId, err)
	}

	return true, nil
}


func (c *VCDClient) RetrieveOrg(vcdname string) (Org, error) {

	req := c.Client.NewRequest(map[string]string{}, "GET", c.OrgHREF, nil)
	req.Header.Add("Accept", "application/*+xml;version=5.5")

	// TODO: wrap into checkresp to parse error
	resp, err := checkResp(c.Client.Http.Do(req))
	if err != nil {
		return Org{}, fmt.Errorf("error retreiving org: %s", err)
	}

	org := NewOrg(&c.Client)

	if err = decodeBody(resp, org.Org); err != nil {
		return Org{}, fmt.Errorf("error decoding org response: %s", err)
	}

	// Get the VDC ref from the Org
	for _, s := range org.Org.Link {
		if s.Type == "application/vnd.vmware.vcloud.vdc+xml" && s.Rel == "down" {
			if vcdname != "" && s.Name != vcdname {
				continue
			}
			u, err := url.Parse(s.HREF)
			if err != nil {
				return Org{}, err
			}
			c.Client.VCDVDCHREF = *u
		}
	}

	if &c.Client.VCDVDCHREF == nil {
		return Org{}, fmt.Errorf("error finding the organization VDC HREF")
	}

	return *org, nil
}

func NewVCDClient(vcdEndpoint url.URL, insecure bool) *VCDClient {

	return &VCDClient{
		Client: Client{
			APIVersion: "5.5",
			VCDVDCHREF: vcdEndpoint,
			Http: http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: insecure,
					},
					Proxy:               http.ProxyFromEnvironment,
					TLSHandshakeTimeout: 120 * time.Second,
				},
			},
		},
	}
}

// Authenticate is an helper function that performs a login in vCloud Director.
func (c *VCDClient) Authenticate(username, password, org, vdcname string) (Org, Vdc, error) {

	// LoginUrl
	err := c.vcdloginurl()
	if err != nil {
		return Org{}, Vdc{}, fmt.Errorf("error finding LoginUrl: %s", err)
	}
	// Authorize
	err = c.vcdauthorize(username, password, org)
	if err != nil {
		return Org{}, Vdc{}, fmt.Errorf("error authorizing: %s", err)
	}

	// Get Org
	o, err := c.RetrieveOrg(vdcname)
	if err != nil {
		return Org{}, Vdc{}, fmt.Errorf("error acquiring Org: %s", err)
	}

	vdc, err := c.Client.retrieveVDC()

	if err != nil {
		return Org{}, Vdc{}, fmt.Errorf("error retrieving the organization VDC: %s : %s ", c.Client.VCDVDCHREF.Path, c.OrgHREF.Path)
	}

	return o, vdc, nil
}

// Disconnect performs a disconnection from the vCloud Director API endpoint.
func (c *VCDClient) Disconnect() error {
	if c.Client.VCDToken == "" && c.Client.VCDAuthHeader == "" {
		return fmt.Errorf("cannot disconnect, client is not authenticated")
	}

	req := c.Client.NewRequest(map[string]string{}, "DELETE", c.sessionHREF, nil)

	// Add the Accept header for vCA
	req.Header.Add("Accept", "application/xml;version=5.5")

	// Set Authorization Header
	req.Header.Add(c.Client.VCDAuthHeader, c.Client.VCDToken)

	if _, err := checkResp(c.Client.Http.Do(req)); err != nil {
		return fmt.Errorf("error processing session delete for vCloud Director: %s", err)
	}
	return nil
}
