package govcloudair

import (
	//"fmt"
	"testing"
	
)


func TestRetrieveVDC(t *testing.T) {
	g, err := GetConfigStruct()
	vcdClient, err := GetTestVCDFromYaml(g)
	if err != nil {
		t.Errorf("Error retrieving vcd client: %v", err)
	}
	_, _, err = vcdClient.Authenticate(g.User, g.Password, g.Orgname, g.Vdcname, true)
	if err != nil {
		t.Errorf("Could not authenticate with user %s password %s url %s: %v", g.User, g.Password, vcdClient.sessionHREF.Path, err)
		t.Errorf("orgname : %s, vcdname : %s", g.Orgname, g.Vdcname)
	}

	vdc , err  := GetVDCFromName(vcdClient, g.Vdcname, g.Orgname) 
	if err != nil {
		t.Errorf("Error retrieving vdc: %v", err)
	}
	if vdc.Vdc.Name != g.Vdcname {
		t.Errorf("Got Wrong VDC: %v", err)
	}
}