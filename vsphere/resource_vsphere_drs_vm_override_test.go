package vsphere

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/structure"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/viapi"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere/internal/helper/virtualmachine"
	"github.com/vmware/govmomi/vim25/types"
)

func TestAccResourceVSphereDRSVMOverride_drs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDRSVMOverridePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDRSVMOverrideExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDRSVMOverrideConfigOverrideDRSEnabled(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDRSVMOverrideExists(true),
					testAccResourceVSphereDRSVMOverrideMatch(types.DrsBehaviorManual, false),
				),
			},
			{
				ResourceName:      "vsphere_drs_vm_override.drs_vm_override",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					cluster, err := testGetComputeClusterFromDataSource(s, "cluster")
					if err != nil {
						return "", err
					}
					vm, err := testGetVirtualMachine(s, "vm")
					if err != nil {
						return "", err
					}

					m := make(map[string]string)
					m["compute_cluster_path"] = cluster.InventoryPath
					m["virtual_machine_path"] = vm.InventoryPath
					b, err := json.Marshal(m)
					if err != nil {
						return "", err
					}

					return string(b), nil
				},
				Config: testAccResourceVSphereDRSVMOverrideConfigOverrideDRSEnabled(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDRSVMOverrideExists(true),
					testAccResourceVSphereDRSVMOverrideMatch(types.DrsBehaviorManual, false),
				),
			},
		},
	})
}

func TestAccResourceVSphereDRSVMOverride_automationLevel(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDRSVMOverridePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDRSVMOverrideExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDRSVMOverrideConfigOverrideAutomationLevel(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDRSVMOverrideExists(true),
					testAccResourceVSphereDRSVMOverrideMatch(types.DrsBehaviorFullyAutomated, true),
				),
			},
		},
	})
}

func TestAccResourceVSphereDRSVMOverride_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccResourceVSphereDRSVMOverridePreCheck(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccResourceVSphereDRSVMOverrideExists(false),
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVSphereDRSVMOverrideConfigOverrideDRSEnabled(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDRSVMOverrideExists(true),
					testAccResourceVSphereDRSVMOverrideMatch(types.DrsBehaviorManual, false),
				),
			},
			{
				Config: testAccResourceVSphereDRSVMOverrideConfigOverrideAutomationLevel(),
				Check: resource.ComposeTestCheckFunc(
					testAccResourceVSphereDRSVMOverrideExists(true),
					testAccResourceVSphereDRSVMOverrideMatch(types.DrsBehaviorFullyAutomated, true),
				),
			},
		},
	})
}

func testAccResourceVSphereDRSVMOverridePreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_DATACENTER") == "" {
		t.Skip("set VSPHERE_DATACENTER to run vsphere_drs_vm_override acceptance tests")
	}
	if os.Getenv("VSPHERE_DATASTORE") == "" {
		t.Skip("set VSPHERE_DATASTORE to run vsphere_drs_vm_override acceptance tests")
	}
	if os.Getenv("VSPHERE_CLUSTER") == "" {
		t.Skip("set VSPHERE_CLUSTER to run vsphere_drs_vm_override acceptance tests")
	}
	if os.Getenv("VSPHERE_NETWORK_LABEL_PXE") == "" {
		t.Skip("set VSPHERE_NETWORK_LABEL_PXE to run vsphere_drs_vm_override acceptance tests")
	}
}

func testAccResourceVSphereDRSVMOverrideExists(expected bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		info, err := testGetComputeClusterDRSVMConfig(s, "drs_vm_override")
		if err != nil {
			if expected == false {
				switch {
				case viapi.IsManagedObjectNotFoundError(err):
					fallthrough
				case virtualmachine.IsUUIDNotFoundError(err):
					// This is not necessarily a missing override, but more than likely a
					// missing cluster, which happens during destroy as the dependent
					// resources will be missing as well, so want to treat this as a
					// deleted override as well.
					return nil
				}
			}
			return err
		}

		switch {
		case info == nil && !expected:
			// Expected missing
			return nil
		case info == nil && expected:
			// Expected to exist
			return errors.New("DRS VM override missing when expected to exist")
		case !expected:
			return errors.New("DRS VM override still present when expected to be missing")
		}

		return nil
	}
}

func testAccResourceVSphereDRSVMOverrideMatch(behavior types.DrsBehavior, enabled bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		actual, err := testGetComputeClusterDRSVMConfig(s, "drs_vm_override")
		if err != nil {
			return err
		}

		if actual == nil {
			return errors.New("DRS VM override missing")
		}

		expected := &types.ClusterDrsVmConfigInfo{
			Behavior: behavior,
			Enabled:  structure.BoolPtr(enabled),
			Key:      actual.Key,
		}

		if !reflect.DeepEqual(expected, actual) {
			return spew.Errorf("expected %#v got %#v", expected, actual)
		}

		return nil
	}
}

func testAccResourceVSphereDRSVMOverrideConfigOverrideDRSEnabled() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

	wait_for_guest_net_timeout = -1

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}

resource "vsphere_drs_vm_override" "drs_vm_override" {
  compute_cluster_id = "${data.vsphere_compute_cluster.cluster.id}"
  virtual_machine_id = "${vsphere_virtual_machine.vm.id}"
  drs_enabled        = false
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}

func testAccResourceVSphereDRSVMOverrideConfigOverrideAutomationLevel() string {
	return fmt.Sprintf(`
variable "datacenter" {
  default = "%s"
}

variable "datastore" {
  default = "%s"
}

variable "cluster" {
  default = "%s"
}

variable "network_label" {
  default = "%s"
}

data "vsphere_datacenter" "dc" {
  name = "${var.datacenter}"
}

data "vsphere_datastore" "datastore" {
  name          = "${var.datastore}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_compute_cluster" "cluster" {
  name          = "${var.cluster}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

data "vsphere_network" "network" {
  name          = "${var.network_label}"
  datacenter_id = "${data.vsphere_datacenter.dc.id}"
}

resource "vsphere_virtual_machine" "vm" {
  name             = "terraform-test"
  resource_pool_id = "${data.vsphere_compute_cluster.cluster.resource_pool_id}"
  datastore_id     = "${data.vsphere_datastore.datastore.id}"

  num_cpus = 2
  memory   = 2048
  guest_id = "other3xLinux64Guest"

	wait_for_guest_net_timeout = -1

  network_interface {
    network_id = "${data.vsphere_network.network.id}"
  }

  disk {
    label = "disk0"
    size  = 20
  }
}

resource "vsphere_drs_vm_override" "drs_vm_override" {
  compute_cluster_id   = "${data.vsphere_compute_cluster.cluster.id}"
  virtual_machine_id   = "${vsphere_virtual_machine.vm.id}"
  drs_enabled          = true
  drs_automation_level = "fullyAutomated"
}
`,
		os.Getenv("VSPHERE_DATACENTER"),
		os.Getenv("VSPHERE_DATASTORE"),
		os.Getenv("VSPHERE_CLUSTER"),
		os.Getenv("VSPHERE_NETWORK_LABEL_PXE"),
	)
}
