package main

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/dns/mgmt/2018-03-01-preview/dns"
)

type Mark string

const (
	Create Mark = "Create"
	Update Mark = "Update"
	Delete Mark = "Delete"
)

var nsrecords = []string{}

type RecordSet struct {
	ZoneName   string     `yaml:"-"`
	Name       string     `yaml:"Name"`
	Type       string     `yaml:"Type"`
	Properties Properties `yaml:"Properties"`
	Mark       Mark       `yaml:"-"`
}

type Properties struct {
	TTL    int      `yaml:"TTL"`
	Values []string `yaml:"Values,omitempty"`
}

// Create new local RecordSet from dns.RecordSet
func NewRecordSet(v dns.RecordSet) (*RecordSet, error) {
	r := RecordSet{}
	r.Name = *v.Name
	r.Type = strings.Replace(*v.Type, "Microsoft.Network/dnszones/", "", -1)
	r.Mark = ""
	r.Properties.TTL = int(*(*v.RecordSetProperties).TTL)

	// r.Properties.Values is empty, need to be initialized.
	// I prefer doing so in each switch/case sentence.
	switch r.Type {
	case "A":
		for _, v := range *v.RecordSetProperties.ARecords {
			r.Properties.Values = append(r.Properties.Values, *v.Ipv4Address)
		}
	case "AAAA":
		for _, v := range *v.RecordSetProperties.AaaaRecords {
			r.Properties.Values = append(r.Properties.Values, *v.Ipv6Address)
		}
	case "CNAME":
		r.Properties.Values = append(r.Properties.Values, *v.RecordSetProperties.CnameRecord.Cname)
	case "MX":
		for _, v := range *v.RecordSetProperties.MxRecords {
			pref := strconv.FormatInt(int64(*v.Preference), 10)
			r.Properties.Values = append(r.Properties.Values, pref+" "+*v.Exchange)
		}
	case "NS":
		for _, v := range *v.RecordSetProperties.NsRecords {
			// Append to the golbal variable
			nsrecords = append(nsrecords, *v.Nsdname)
		}
	case "TXT":
		for _, v := range *v.RecordSetProperties.TxtRecords {
			r.Properties.Values = append(r.Properties.Values, "")
			for _, s := range *v.Value {
				r.Properties.Values[0] += s
			}
		}
	default:
		return nil, nil
	}

	return &r, nil
}

func (r *RecordSet) splitSubN(s string, n int) []string {
	sub := ""
	subs := []string{}

	runes := []rune(s)
	l := len(runes)

	for i, r := range runes {
		sub = sub + string(r)
		if (1+i)%n == 0 {
			subs = append(subs, sub)
			sub = ""
		} else if (i + 1) == l {
			subs = append(subs, sub)
		}
	}

	return subs
}

func (r *RecordSet) sha1Sum() string {
	values := ""
	for _, v := range r.Properties.Values {
		values += v
	}
	data := r.Name + r.Type + strconv.Itoa(r.Properties.TTL) + values
	h := sha1.New()
	io.WriteString(h, data)

	str := fmt.Sprintf("%x", h.Sum(nil))
	return str
}

func (r *RecordSet) createOrUpdate() (*dns.RecordSet, error) {
	rsc := dns.NewRecordSetsClient(session.SubscriptionID)
	rsc.Authorizer = session.Authorizer

	recordSet := dns.RecordSet{}
	ttl := int64(r.Properties.TTL)
	recordSet.RecordSetProperties = &dns.RecordSetProperties{}
	recordSet.RecordSetProperties.TTL = &ttl

	switch r.Type {
	case "A":
		records := []dns.ARecord{}
		for i, _ := range r.Properties.Values {
			// Don't use _, v here because `range` copies the values and &v doesn't work
			val := r.Properties.Values[i]
			record := dns.ARecord{}
			record.Ipv4Address = &val
			records = append(records, record)
		}
		recordSet.RecordSetProperties.ARecords = &records
	case "AAAA":
		records := []dns.AaaaRecord{}
		for i, _ := range r.Properties.Values {
			val := r.Properties.Values[i]
			record := dns.AaaaRecord{}
			record.Ipv6Address = &val
			records = append(records, record)
		}
		recordSet.RecordSetProperties.AaaaRecords = &records
	case "CNAME":
		if len(r.Properties.Values) != 1 {
			return nil, errors.New("Invalid cname records")
		}
		cname := dns.CnameRecord{}
		cname.Cname = &r.Properties.Values[0]
		recordSet.RecordSetProperties.CnameRecord = &cname
	case "MX":
		records := []dns.MxRecord{}
		for _, v := range r.Properties.Values {
			record := dns.MxRecord{}

			vals := strings.Split(v, " ")
			i, err := strconv.Atoi(vals[0])
			if err != nil {
				return nil, err
			}
			i32 := int32(i)
			record.Preference = &i32
			record.Exchange = &vals[1]
			records = append(records, record)
		}
		recordSet.RecordSetProperties.MxRecords = &records
	case "NS":
		records := []dns.NsRecord{}
		// Read from the global variable
		for i, _ := range nsrecords {
			record := dns.NsRecord{}
			val := nsrecords[i]
			record.Nsdname = &val
			records = append(records, record)
		}
		recordSet.RecordSetProperties.NsRecords = &records
	case "TXT":
		// here
		if len(r.Properties.Values) != 1 {
			return nil, errors.New("Invalid txt records")
		}
		records := []dns.TxtRecord{}

		vals := r.splitSubN(r.Properties.Values[0], 255)
		if len(vals) == 0 {
			return nil, errors.New("Invalid txt records")
		}
		record := dns.TxtRecord{}
		record.Value = &vals

		records = append(records, record)
		recordSet.RecordSetProperties.TxtRecords = &records
	default:
		// We don't handle CAA, PTR, SOA and SRV records
		// Just return nil to let the loop continue
		return nil, nil
	}

	result, err := rsc.CreateOrUpdate(context.Background(), ResourceGroupName, r.ZoneName, r.Name, dns.RecordType(r.Type), recordSet, "", "")
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *RecordSet) delete() error {
	r.message()

	rsc := dns.NewRecordSetsClient(session.SubscriptionID)
	rsc.Authorizer = session.Authorizer

	result, err := rsc.Delete(context.Background(), ResourceGroupName, r.ZoneName, r.Name, dns.RecordType(r.Type), "")
	if err != nil {
		return err
	}

	if result.StatusCode == 200 {
		fmt.Printf("Deleted %v on %v\n", r.Name, r.Type)
	} else {
		return errors.New(result.Status)
	}

	return nil
}

func (r *RecordSet) create() error {
	r.message()

	result, err := r.createOrUpdate()
	if err != nil {
		return err
	}

	fmt.Printf("Created %v on %v\n", *result.Name, *result.Type)

	return nil
}

func (r *RecordSet) update() error {
	r.message()

	result, err := r.createOrUpdate()
	if err != nil {
		return err
	}

	fmt.Printf("Updated %v on %v\n", *result.Name, *result.Type)

	return nil
}

func (r *RecordSet) message() {
	var verb string

	switch r.Mark {
	case Delete:
		verb = "deleted"
	case Create:
		verb = "created"
	case Update:
		verb = "updated"
	default:
		verb = ""
	}

	fmt.Printf("%v on %v will be "+verb+". Values:\n", r.Name, r.Type)
	fmt.Printf("    TTL: %v\n", r.Properties.TTL)
	for _, v := range r.Properties.Values {
		fmt.Printf("    %v\n", v)
	}
}
