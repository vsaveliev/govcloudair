/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	types "github.com/vsaveliev/govcloudair/types/v56"
	"log"
	"net/url"
	"os"
	"strings"
)

type Vdc struct {
	Vdc  *types.Vdc
	c    *Client
	VApp *types.VApp
}

func NewVdc(c *Client) *Vdc {
	return &Vdc{
		Vdc: new(types.Vdc),
		c:   c,
	}
}

func (c *Client) retrieveVDC() (Vdc, error) {

	req := c.NewRequest(map[string]string{}, "GET", c.VCDVDCHREF, nil)

	resp, err := checkResp(c.Http.Do(req))
	if err != nil {
		return Vdc{}, fmt.Errorf("error retreiving vdc: %s", err)
	}

	vdc := NewVdc(c)

	if err = decodeBody(resp, vdc.Vdc); err != nil {
		return Vdc{}, fmt.Errorf("error decoding vdc response: %s", err)
	}

	// The request was successful
	return *vdc, nil
}

func (v *Vdc) Refresh() error {

	if v.Vdc.HREF == "" {
		return fmt.Errorf("cannot refresh, Object is empty")
	}

	u, _ := url.ParseRequestURI(v.Vdc.HREF)

	req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retreiving Edge Gateway: %s", err)
	}

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	unmarshalledVdc := &types.Vdc{}

	if err = decodeBody(resp, unmarshalledVdc); err != nil {
		return fmt.Errorf("error decoding vdc response: %s", err)
	}

	v.Vdc = unmarshalledVdc

	// The request was successful
	return nil
}

func (v *Vdc) FindVDCNetwork(network string) (OrgVDCNetwork, error) {

	for _, an := range v.Vdc.AvailableNetworks {
		for _, n := range an.Network {
			if n.Name == network {
				u, err := url.ParseRequestURI(n.HREF)
				if err != nil {
					return OrgVDCNetwork{}, fmt.Errorf("error decoding vdc response: %s", err)
				}

				req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

				resp, err := checkResp(v.c.Http.Do(req))
				if err != nil {
					return OrgVDCNetwork{}, fmt.Errorf("error retreiving orgvdcnetwork: %s", err)
				}

				orgnet := NewOrgVDCNetwork(v.c)

				if err = decodeBody(resp, orgnet.OrgVDCNetwork); err != nil {
					return OrgVDCNetwork{}, fmt.Errorf("error decoding orgvdcnetwork response: %s", err)
				}

				// The request was successful
				return *orgnet, nil

			}
		}
	}

	return OrgVDCNetwork{}, fmt.Errorf("can't find VDC Network: %s", network)
}

func (v *Vdc) FindStorageProfileReference(name string) (types.Reference, error) {

	for _, sps := range v.Vdc.VdcStorageProfiles {
		for _, sp := range sps.VdcStorageProfile {
			if sp.Name == name {
				return types.Reference{HREF: sp.HREF, Name: sp.Name}, nil
			}
		}
		return types.Reference{}, fmt.Errorf("can't find VDC Storage_profile: %s", name)
	}
	return types.Reference{}, fmt.Errorf("can't find any VDC Storage_profiles")
}

func (v *Vdc) GetDefaultStorageProfileReference(storageprofiles *types.QueryResultRecordsType) (types.Reference, error) {

	for _, spr := range storageprofiles.OrgVdcStorageProfileRecord {
		if spr.IsDefaultStorageProfile {
			return types.Reference{HREF: spr.HREF, Name: spr.Name}, nil
		}
	}
	return types.Reference{}, fmt.Errorf("can't find Default VDC Storage_profile")
}

// Doesn't work with vCloud API 5.5, only vCloud Air
func (v *Vdc) GetVDCOrg() (Org, error) {

	for _, av := range v.Vdc.Link {
		if av.Rel == "up" && av.Type == "application/vnd.vmware.vcloud.org+xml" {
			u, err := url.ParseRequestURI(av.HREF)

			if err != nil {
				return Org{}, fmt.Errorf("error decoding vdc response: %s", err)
			}

			req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err := checkResp(v.c.Http.Do(req))
			if err != nil {
				return Org{}, fmt.Errorf("error retreiving org: %s", err)
			}

			org := NewOrg(v.c)

			if err = decodeBody(resp, org.Org); err != nil {
				return Org{}, fmt.Errorf("error decoding org response: %s", err)
			}

			// The request was successful
			return *org, nil

		}
	}
	return Org{}, fmt.Errorf("can't find VDC Org")
}

func (v *Vdc) FindEdgeGateway(edgegateway string) (EdgeGateway, error) {

	for _, av := range v.Vdc.Link {
		if av.Rel == "edgeGateways" && av.Type == "application/vnd.vmware.vcloud.query.records+xml" {
			u, err := url.ParseRequestURI(av.HREF)

			if err != nil {
				return EdgeGateway{}, fmt.Errorf("error decoding vdc response: %s", err)
			}

			// Querying the Result list
			req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err := checkResp(v.c.Http.Do(req))
			if err != nil {
				return EdgeGateway{}, fmt.Errorf("error retrieving edge gateway records: %s", err)
			}

			query := new(types.QueryResultEdgeGatewayRecordsType)

			if err = decodeBody(resp, query); err != nil {
				return EdgeGateway{}, fmt.Errorf("error decoding edge gateway query response: %s", err)
			}

			u, err = url.ParseRequestURI(query.EdgeGatewayRecord.HREF)
			if err != nil {
				return EdgeGateway{}, fmt.Errorf("error decoding edge gateway query response: %s", err)
			}

			// Querying the Result list
			req = v.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err = checkResp(v.c.Http.Do(req))
			if err != nil {
				return EdgeGateway{}, fmt.Errorf("error retrieving edge gateway: %s", err)
			}

			edge := NewEdgeGateway(v.c)

			if err = decodeBody(resp, edge.EdgeGateway); err != nil {
				return EdgeGateway{}, fmt.Errorf("error decoding edge gateway response: %s", err)
			}

			return *edge, nil

		}
	}
	return EdgeGateway{}, fmt.Errorf("can't find Edge Gateway")

}

func (v *Vdc) ComposeRawVApp(name string) error {
	vcomp := &types.ComposeVAppParams{
		Ovf:     "http://schemas.dmtf.org/ovf/envelope/1",
		Xsi:     "http://www.w3.org/2001/XMLSchema-instance",
		Xmlns:   "http://www.vmware.com/vcloud/v1.5",
		Deploy:  false,
		Name:    name,
		PowerOn: false,
	}

	output, err := xml.MarshalIndent(vcomp, "  ", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling vapp compose: %s", err)
	}

	debug := os.Getenv("GOVCLOUDAIR_DEBUG")

	if debug == "true" {
		fmt.Printf("\n\nXML DEBUG: %s\n\n", string(output))
	}

	b := bytes.NewBufferString(xml.Header + string(output))

	s := v.c.VCDVDCHREF
	s.Path += "/action/composeVApp"

	req := v.c.NewRequest(map[string]string{}, "POST", s, b)

	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.composeVAppParams+xml")

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error instantiating a new vApp: %s", err)
	}

	task := NewTask(v.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return fmt.Errorf("error decoding task response: %s", err)
	}

	err = task.WaitTaskCompletion()
	if err != nil {
		return fmt.Errorf("Error performing task: %#v", err)
	}

	return nil
}

func (v *Vdc) FindVAppByName(vapp string) (VApp, error) {

	err := v.Refresh()
	if err != nil {
		return VApp{}, fmt.Errorf("error refreshing vdc: %s", err)
	}

	for _, resents := range v.Vdc.ResourceEntities {
		for _, resent := range resents.ResourceEntity {

			if resent.Name == vapp && resent.Type == "application/vnd.vmware.vcloud.vApp+xml" {

				u, err := url.ParseRequestURI(resent.HREF)

				if err != nil {
					return VApp{}, fmt.Errorf("error decoding vdc response: %s", err)
				}

				// Querying the VApp
				req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

				resp, err := checkResp(v.c.Http.Do(req))
				if err != nil {
					return VApp{}, fmt.Errorf("error retrieving vApp: %s", err)
				}

				newvapp := NewVApp(v.c)

				if err = decodeBody(resp, newvapp.VApp); err != nil {
					return VApp{}, fmt.Errorf("error decoding vApp response: %s", err.Error())
				}

				return *newvapp, nil

			}
		}
	}
	return VApp{}, fmt.Errorf("can't find vApp: %s", vapp)
}

func (v *Vdc) FindVMByName(vapp VApp, vm string) (VM, error) {

	err := v.Refresh()
	if err != nil {
		return VM{}, fmt.Errorf("error refreshing vdc: %s", err)
	}

	err = vapp.Refresh()
	if err != nil {
		return VM{}, fmt.Errorf("error refreshing vapp: %s", err)
	}

	log.Printf("[TRACE] Looking for VM: %s", vm)
	for _, child := range vapp.VApp.Children.VM {

		log.Printf("[TRACE] Found: %s", child.Name)
		if child.Name == vm {

			u, err := url.ParseRequestURI(child.HREF)

			if err != nil {
				return VM{}, fmt.Errorf("error decoding vdc response: %s", err)
			}

			// Querying the VApp
			req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

			resp, err := checkResp(v.c.Http.Do(req))
			if err != nil {
				return VM{}, fmt.Errorf("error retrieving vm: %s", err)
			}

			newvm := NewVM(v.c)

			//body, err := ioutil.ReadAll(resp.Body)
			//fmt.Println(string(body))

			if err = decodeBody(resp, newvm.VM); err != nil {
				return VM{}, fmt.Errorf("error decoding vm response: %s", err.Error())
			}

			return *newvm, nil

		}

	}
	log.Printf("[TRACE] Couldn't find VM: %s", vm)
	return VM{}, fmt.Errorf("can't find vm: %s", vm)
}

func (v *Vdc) FindVAppByID(vappid string) (VApp, error) {

	// Horrible hack to fetch a vapp with its id.
	// urn:vcloud:vapp:00000000-0000-0000-0000-000000000000

	err := v.Refresh()
	if err != nil {
		return VApp{}, fmt.Errorf("error refreshing vdc: %s", err)
	}

	urnslice := strings.SplitAfter(vappid, ":")
	urnid := urnslice[len(urnslice)-1]

	for _, resents := range v.Vdc.ResourceEntities {
		for _, resent := range resents.ResourceEntity {

			hrefslice := strings.SplitAfter(resent.HREF, "/")
			hrefslice = strings.SplitAfter(hrefslice[len(hrefslice)-1], "-")
			res := strings.Join(hrefslice[1:], "")

			if res == urnid && resent.Type == "application/vnd.vmware.vcloud.vApp+xml" {

				u, err := url.ParseRequestURI(resent.HREF)

				if err != nil {
					return VApp{}, fmt.Errorf("error decoding vdc response: %s", err)
				}

				// Querying the VApp
				req := v.c.NewRequest(map[string]string{}, "GET", *u, nil)

				resp, err := checkResp(v.c.Http.Do(req))
				if err != nil {
					return VApp{}, fmt.Errorf("error retrieving vApp: %s", err)
				}

				newvapp := NewVApp(v.c)

				if err = decodeBody(resp, newvapp.VApp); err != nil {
					return VApp{}, fmt.Errorf("error decoding vApp response: %s", err)
				}

				return *newvapp, nil

			}
		}
	}
	return VApp{}, fmt.Errorf("can't find vApp")

}
