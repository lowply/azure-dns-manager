package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/services/preview/dns/mgmt/2018-03-01-preview/dns"
	"github.com/mitchellh/hashstructure"
	"gopkg.in/yaml.v2"
)

type Zone struct {
	Name       string      `yaml:"Zone"`
	RecordSets []RecordSet `yaml:"RecordSets"`
}

func NewZone(name string, remote bool) (*Zone, error) {
	if name == "" {
		return nil, errors.New("Need a name")
	}
	z := Zone{}
	z.Name = name

	if remote {
		err := z.readFromRemote()
		if err != nil {
			return nil, err
		}
	} else {
		err := z.readFromFile()
		if err != nil {
			return nil, err
		}
	}

	return &z, nil
}

func (z *Zone) readFromRemote() error {
	var top int32 = 30

	rsc := dns.NewRecordSetsClient(session.SubscriptionID)
	rsc.Authorizer = session.Authorizer

	recordsets, err := rsc.ListByDNSZone(context.Background(), ResourceGroupName, z.Name, &top, "")
	if err != nil {
		return err
	}

	// Values() returns []RecordSet
	for _, v := range recordsets.Values() {
		r, err := NewRecordSet(v)
		if err != nil {
			return err
		}

		// Skip append if the type is one of PTR, SOA and SRV.
		if r == nil {
			continue
		}

		z.RecordSets = append(z.RecordSets, *r)
	}

	return nil
}

func (z *Zone) readFromFile() error {
	filepath := filepath.Join(azure_dns_zones, z.Name+".yaml")

	file, err := os.Open(filepath)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &z)
	if err != nil {
		return err
	}

	return nil
}

func (z *Zone) markDelete(t *Zone) (int, error) {
	count := 0
R:
	for i, v := range z.RecordSets {
		for _, k := range t.RecordSets {
			if v.Name == k.Name && v.Type == k.Type {
				continue R
			}
		}
		z.RecordSets[i].Mark = Delete
		z.RecordSets[i].ZoneName = z.Name
		count++
	}
	return count, nil
}

func (z *Zone) markCreate(t *Zone) (int, error) {
	count := 0
R:
	for i, v := range z.RecordSets {
		for _, k := range t.RecordSets {
			if v.Name == k.Name && v.Type == k.Type {
				continue R
			}
		}
		z.RecordSets[i].Mark = Create
		z.RecordSets[i].ZoneName = z.Name
		count++
	}
	return count, nil
}

func (z *Zone) markUpdate(t *Zone) (int, error) {
	count := 0
R:
	for i, v := range z.RecordSets {
		if v.Mark == Create {
			continue
		}
		hashv, err := hashstructure.Hash(v, nil)
		if err != nil {
			return 0, err
		}
		for _, k := range t.RecordSets {
			hashk, err := hashstructure.Hash(k, nil)
			if err != nil {
				return 0, err
			}
			if hashv == hashk {
				continue R
			}
		}
		z.RecordSets[i].Mark = Update
		z.RecordSets[i].ZoneName = z.Name
		count++
	}
	return count, nil
}

func (z *Zone) syncRecordSets(remote *Zone) error {

	// mark Delete first
	cd, err := remote.markDelete(z)
	if err != nil {
		return err
	}

	// mark Create next
	cc, err := z.markCreate(remote)
	if err != nil {
		return err
	}

	// Finally, mark Update for the rest of the records
	cu, err := z.markUpdate(remote)
	if err != nil {
		return err
	}

	if cd+cc+cu == 0 {
		fmt.Println("No change")
		return nil
	}

	// Delete first
	for _, r := range remote.RecordSets {
		if r.Mark == Delete {
			err := r.delete()
			if err != nil {
				return err
			}
		}
	}

	// Create next
	for _, r := range z.RecordSets {
		if r.Mark == Create {
			err := r.create()
			if err != nil {
				return err
			}
		}
	}

	// Update at last
	for _, r := range z.RecordSets {
		if r.Mark == Update {
			err := r.update()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
