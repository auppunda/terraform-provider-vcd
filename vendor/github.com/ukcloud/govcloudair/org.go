/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"fmt"
	"net/url"

	"strings"
	types "github.com/ukcloud/govcloudair/types/v56"
)

type Org struct {
	Org *types.Org
	c   *Client
}

func NewOrg(c *Client) *Org {
	return &Org{
		Org: new(types.Org),
		c:   c,
	}
}

func (o *Org) RemoveAllOrgVDCs(href url.URL) (Task, error){
	var task *Task
	//_, org, _ := c.GetOrg(orgId)
	count := 0
	for _, a := range o.Org.Link {
		if a.Type == "application/vnd.vmware.vcloud.vdc+xml" && a.Rel == "down" {
			u, err := url.Parse(a.HREF)
			if err != nil {
				return Task{} , err
			}
				
			req := o.c.NewRequest(map[string]string{}, "GET", *u , nil)

			resp, err := checkResp(o.c.Http.Do(req))
			if err != nil {
				return Task{}, fmt.Errorf("error retreiving vdc: %s", err)
			}

			vdc := NewVdc(o.c)

			if err = decodeBody(resp, vdc.Vdc); err != nil {
				return Task{}, fmt.Errorf("error decoding vdc response: %s", err)
			}

			//vdc.DeleteAllVapps()
			s := href
			s.Path += "/admin/vdc/" + vdc.Vdc.ID[15:]

			copyPath := s.Path

			s.Path += "/action/disable"


			req = o.c.NewRequest(map[string]string{}, "POST", s, nil)

			_ , err = checkResp(o.c.Http.Do(req))

			if err != nil {
				return Task{}, fmt.Errorf("error disabling vdc: %s", err)
			}

			s.Path = copyPath

			req = o.c.NewRequest(map[string]string{}, "DELETE", s, nil)	

			resp , err = checkResp(o.c.Http.Do(req))

			if err != nil {
				return Task{}, fmt.Errorf("error deleting vdc: %s", err)
			}

			task = NewTask(o.c)

			if err = decodeBody(resp, task.Task); err != nil {
				return Task{}, fmt.Errorf("error decoding task response: %s", err)
			}			

			if task.Task.Status == "error" {
				return *task, fmt.Errorf("vdc not properly destroyed")
			}

				//c.Client.VCDVDCHREF = *u
			count = count + 1
		}

	}
	if count == 0 {
		return Task{
			Task: &types.Task{},
			}, nil
	}

	return *task, nil
}

//removes All networks in the org
func (o *Org) RemoveAllOrgNetworks(HREF url.URL) (Task, error){
	var task *Task
	//_, org, _ := c.GetOrg(orgId)
	count := 0
	for _, a := range o.Org.Link {
		if a.Type == "application/vnd.vmware.vcloud.orgNetwork+xml" && a.Rel == "down" {
			u, err := url.Parse(a.HREF)
			if err != nil {
				return Task{} , err
			}

			s := HREF
			s.Path += "/admin/network/" + strings.Split(u.Path, "/network/")[1] //gets id

			req := o.c.NewRequest(map[string]string{}, "DELETE", s, nil)	

			resp , err := checkResp(o.c.Http.Do(req))

			if err != nil {
				return Task{}, fmt.Errorf("error deleting newtork: %s, %s", err, u.Path)
			}

			task = NewTask(o.c)

			if err = decodeBody(resp, task.Task); err != nil {
				return Task{}, fmt.Errorf("error decoding task response: %s", err)
			}			

			if task.Task.Status == "error" {
				return *task, fmt.Errorf("vdc not properly destroyed")
			}

				//c.Client.VCDVDCHREF = *u
			count = count + 1
		}

	}
	if count == 0 {
		return Task{
			Task: &types.Task{},
			}, nil
	}

	return *task, nil
}
func (o *Org) FindCatalog(catalog string) (Catalog, error) {

	for _, av := range o.Org.Link {
		if av.Rel == "down" && av.Type == "application/vnd.vmware.vcloud.catalog+xml" && av.Name == catalog {
			u, err := url.ParseRequestURI(av.HREF)

			if err != nil {
				return Catalog{}, fmt.Errorf("error decoding org response: %s", err)
			}

			req := o.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err := checkResp(o.c.Http.Do(req))
			if err != nil {
				return Catalog{}, fmt.Errorf("error retreiving catalog: %s", err)
			}

			cat := NewCatalog(o.c)

			if err = decodeBody(resp, cat.Catalog); err != nil {
				return Catalog{}, fmt.Errorf("error decoding catalog response: %s", err)
			}

			// The request was successful
			return *cat, nil

		}
	}

	return Catalog{}, fmt.Errorf("can't find catalog: %s", catalog)
}
