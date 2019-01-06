package main

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/dns/mgmt/2018-03-01-preview/dns"
)

func TestNewRecordSet(t *testing.T) {
	v := dns.RecordSet{}
	r, err := NewRecordSet(v)
	if err != nil {
		t.Fatal("Failed")
	}
	r.Properties.Values
}

func TestSplitSubN(t *testing.T) {
	r := RecordSet{}
	result := r.splitSubN("ccccccccccbbbbbbbbbboooooooooopppppppppp", 10)
	if len(result) != 4 {
		t.Fatal("Failed")
	}
}
