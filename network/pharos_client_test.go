package network_test

import (
	"encoding/json"
	"fmt"
	"github.com/APTrust/exchange/models"
	"github.com/APTrust/exchange/network"
	"github.com/APTrust/exchange/testdata"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)


func TestNewPharosClient(t *testing.T) {
	_, err := network.NewPharosClient("http://example.com", "v1", "user", "key")
	if err != nil {
		t.Error(err)
	}
}

func TestInstitutionGet(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(institutionGetHander))
	defer testServer.Close()

	client, err := network.NewPharosClient(testServer.URL, "v1", "user", "key")
	if err != nil {
		t.Error(err)
		return
	}

	response := client.InstitutionGet("college.edu")

	// Check the request URL and method
	assert.Equal(t, "GET", response.Response.Request.Method)
	assert.Equal(t, "/api/v1/institutions/college.edu", response.Request.URL.Opaque)

	// Basic sanity check on response values
	assert.Nil(t, response.Error)
	assert.EqualValues(t, "Institution", response.ObjectType())
	if response.Institution() == nil {
		t.Errorf("Institution should not be nil")
	}
	assert.NotEqual(t, "", response.Institution().Identifier)
}

func TestInstitutionList(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(institutionListHander))
	defer testServer.Close()

	client, err := network.NewPharosClient(testServer.URL, "v1", "user", "key")
	if err != nil {
		t.Error(err)
		return
	}

	response := client.InstitutionList()

	// Check the request URL and method
	assert.Equal(t, "GET", response.Response.Request.Method)
	assert.Equal(t, "/api/v1/institutions/", response.Request.URL.Opaque)

	// Basic sanity check on response values
	assert.Nil(t, response.Error)
	assert.EqualValues(t, "Institution", response.ObjectType())

	instList := response.Institutions()
	if instList == nil {
		t.Errorf("Institution list should not be nil")
		return
	}
	if len(instList) != 4 {
		t.Errorf("Institutions list should have four items. Found %d.", len(instList))
		return
	}
	for _, inst := range instList {
		assert.NotEqual(t, "", len(inst.Identifier))
	}
}

func TestIntellectualObjectGet(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(intellectualObjectGetHander))
	defer testServer.Close()

	client, err := network.NewPharosClient(testServer.URL, "v1", "user", "key")
	if err != nil {
		t.Error(err)
		return
	}

	response := client.IntellectualObjectGet("college.edu/object")

	// Check the request URL and method
	assert.Equal(t, "GET", response.Response.Request.Method)
	assert.Equal(t, "/api/v1/objects/college.edu%2Fobject", response.Request.URL.Opaque)

	// Basic sanity check on response values
	assert.Nil(t, response.Error)

	obj := response.IntellectualObject()
	assert.EqualValues(t, "IntellectualObject", response.ObjectType())
	if obj == nil {
		t.Errorf("IntellectualObject should not be nil")
	}
	assert.NotEqual(t, "", obj.Identifier)
	assert.Equal(t, 2, len(obj.GenericFiles))
	assert.Equal(t, 3, len(obj.PremisEvents))
	assert.Equal(t, 4, len(obj.GenericFiles[0].Checksums))
	assert.Equal(t, 5, len(obj.IngestTags))
}


// -------------------------------------------------------------------------

// Build a simple struct that mimics the structure of a Pharos
// JSON list response. That includes keys count, next, previous,
// and results. The caller will add ["results"] with a list of
// objects of the appropriate type.
func listResponseData() (map[string]interface{}) {
	data := make(map[string]interface{})
	data["count"] = 100
	data["next"] = "http://example.com/?page=11"
	data["previous"] = "http://example.com/?page=9"
	return data
}

func institutionGetHander(w http.ResponseWriter, r *http.Request) {
	inst := testdata.MakeInstitution()
	instJson, _ := json.Marshal(inst)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(instJson))
}

func institutionListHander(w http.ResponseWriter, r *http.Request) {
	instList := make([]*models.Institution, 4)
	for i := 0; i < 4; i++ {
		instList[i] = testdata.MakeInstitution()
	}
	data := listResponseData()
	data["results"] = instList
	instJson, _ := json.Marshal(data)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(instJson))
}

func intellectualObjectGetHander(w http.ResponseWriter, r *http.Request) {
	obj := testdata.MakeIntellectualObject(2,3,4,5)
	objJson, _ := json.Marshal(obj)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, string(objJson))
}
