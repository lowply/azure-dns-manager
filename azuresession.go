package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/Azure/azure-sdk-for-go/services/preview/dns/mgmt/2018-03-01-preview/dns"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

const (
	ResourceGroupName     = "azure-dns-manager"
	ResourceGroupLocation = "japaneast"
)

type AzureSession struct {
	SubscriptionID string
	Authorizer     autorest.Authorizer
}

var session *AzureSession

func readJSON(path string) (*map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, errors.Wrap(err, "Can't open the file")
	}

	contents := make(map[string]interface{})
	err = json.Unmarshal(data, &contents)

	if err != nil {
		err = errors.Wrap(err, "Can't unmarshal file")
	}

	return &contents, err
}

func NewAzureSession(auth_file_path string) (*AzureSession, error) {
	authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "Can't initialize authorizer")
	}

	authInfo, err := readJSON(auth_file_path)
	if err != nil {
		return nil, errors.Wrap(err, "Can't get authinfo")
	}

	sess := AzureSession{
		SubscriptionID: (*authInfo)["subscriptionId"].(string),
		Authorizer:     authorizer,
	}

	return &sess, nil
}

func (s *AzureSession) createZone(zonename string) error {
	zc := dns.NewZonesClient(session.SubscriptionID)
	zc.Authorizer = session.Authorizer

	z := dns.Zone{}
	location := "global"
	z.Location = &location
	z.Name = &zonename

	result, err := zc.CreateOrUpdate(context.Background(), ResourceGroupName, zonename, z, "", "")
	if err != nil {
		return err
	}

	fmt.Printf("New zone has been created: %v\n", *result.Name)

	return nil
}

func (s *AzureSession) listZones() (*[]string, error) {
	var top int32 = 30

	zc := dns.NewZonesClient(s.SubscriptionID)
	zc.Authorizer = s.Authorizer

	list, err := zc.List(context.Background(), &top)
	if err != nil {
		return nil, err
	}

	zones := []string{}

	for _, v := range list.Values() {
		zones = append(zones, *v.Name)
	}

	return &zones, nil
}

func (s *AzureSession) checkOrCreateResourceGroup() error {
	name := ResourceGroupName
	location := ResourceGroupLocation

	rgc := resources.NewGroupsClient(s.SubscriptionID)
	rgc.Authorizer = s.Authorizer

	result, err := rgc.CheckExistence(context.Background(), name)
	if err != nil {
		return err
	}

	if result.StatusCode == 404 {
		g := resources.Group{}
		g.Name = &name
		g.Location = &location
		_, err := rgc.CreateOrUpdate(context.Background(), name, g)
		if err != nil {
			return err
		}
		fmt.Printf("Created resource group %v\n", name)
	}

	return nil
}
