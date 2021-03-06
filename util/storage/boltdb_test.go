package storage_test

import (
	"encoding/json"
	"fmt"
	"github.com/APTrust/exchange/models"
	"github.com/APTrust/exchange/util/storage"
	"github.com/APTrust/exchange/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestBoltDB(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "boltdb_test")
	require.Nil(t, err)
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	bolt, err := storage.NewBoltDB(tempFile.Name())
	require.Nil(t, err)
	defer bolt.Close()

	// Save and retrieve an object
	obj := testutil.MakeIntellectualObject(1, 1, 1, 10)
	err = bolt.Save("Test Object", obj)
	require.Nil(t, err)

	restoredObj, err := bolt.GetIntellectualObject("Test Object")
	require.Nil(t, err)
	require.NotNil(t, restoredObj)
	assert.Equal(t, obj.Identifier, restoredObj.Identifier)

	nilObj, err := bolt.GetIntellectualObject("Nil Object")
	require.Nil(t, err)
	require.Nil(t, nilObj)

	// Save and retrieve a generic file
	gfIdentifier := ""
	for i := 0; i < 10; i++ {
		gf := testutil.MakeGenericFile(2, 2, gfIdentifier)
		err = bolt.Save(gf.Identifier, gf)
		require.Nil(t, err)
		gfIdentifier = gf.Identifier
	}

	restoredFile, err := bolt.GetGenericFile(gfIdentifier)
	require.Nil(t, err)
	require.NotNil(t, restoredFile)
	assert.Equal(t, gfIdentifier, restoredFile.Identifier)

	nilFile, err := bolt.GetGenericFile("Nil File")
	require.Nil(t, err)
	require.Nil(t, nilFile)

	// Get a list of GenericFile keys. Should not return obj identifier
	gfIds := bolt.FileIdentifiers()
	require.Equal(t, 10, len(gfIds))

	assert.Equal(t, 10, bolt.FileCount())

	assert.Equal(t, "Test Object", bolt.ObjectIdentifier())
}

func TestBoltDB_FileIdentifierBatch(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "boltdb_test")
	require.Nil(t, err)
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	bolt, err := storage.NewBoltDB(tempFile.Name())
	require.Nil(t, err)
	defer bolt.Close()

	// Save and retrieve an object
	obj := testutil.MakeIntellectualObject(1, 1, 1, 10)
	err = bolt.Save("Test Object", obj)
	require.Nil(t, err)

	// Put in 20 objects
	for i := 0; i < 20; i++ {
		gfId := fmt.Sprintf("uc.edu/bag/data/file_%02d.json", i)
		gf := testutil.MakeGenericFile(2, 2, gfId)
		err = bolt.Save(gfId, gf)
		require.Nil(t, err)
	}

	batch := bolt.FileIdentifierBatch(0, 5)
	assert.Equal(t, 5, len(batch))
	assert.Equal(t, "uc.edu/bag/data/file_00.json", batch[0])
	assert.Equal(t, "uc.edu/bag/data/file_04.json", batch[4])

	batch = bolt.FileIdentifierBatch(5, 5)
	assert.Equal(t, 5, len(batch))
	assert.Equal(t, "uc.edu/bag/data/file_05.json", batch[0])
	assert.Equal(t, "uc.edu/bag/data/file_09.json", batch[4])

	batch = bolt.FileIdentifierBatch(10, 5)
	assert.Equal(t, 5, len(batch))
	assert.Equal(t, "uc.edu/bag/data/file_10.json", batch[0])
	assert.Equal(t, "uc.edu/bag/data/file_14.json", batch[4])

	batch = bolt.FileIdentifierBatch(15, 5)
	assert.Equal(t, 5, len(batch))
	assert.Equal(t, "uc.edu/bag/data/file_15.json", batch[0])
	assert.Equal(t, "uc.edu/bag/data/file_19.json", batch[4])

	batch = bolt.FileIdentifierBatch(20, 5)
	assert.Equal(t, 0, len(batch))

	batch = bolt.FileIdentifierBatch(-100, -20)
	assert.Equal(t, 0, len(batch))
}

func TestBoltDB_DumpJson(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "boltdb_test")
	require.Nil(t, err)
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	bolt, err := storage.NewBoltDB(tempFile.Name())
	require.Nil(t, err)
	defer bolt.Close()

	// Save an object
	obj := testutil.MakeIntellectualObject(1, 1, 1, 10)
	err = bolt.Save("Test Object", obj)
	require.Nil(t, err)

	// Add some files
	for i := 0; i < 20; i++ {
		gfId := fmt.Sprintf("uc.edu/bag/data/file_%02d.json", i)
		gf := testutil.MakeGenericFile(2, 2, gfId)
		err = bolt.Save(gfId, gf)
		require.Nil(t, err)
	}

	// Dump Json into a file
	jsonFile, err := ioutil.TempFile("", "boltdb_test_json")
	if jsonFile != nil {
		defer jsonFile.Close()
	}
	require.Nil(t, err)
	err = bolt.DumpJson(jsonFile)
	require.Nil(t, err)

	// Rewind and read the file
	_, err = jsonFile.Seek(0, io.SeekStart)
	require.Nil(t, err)
	jsonBytes, err := ioutil.ReadAll(jsonFile)
	require.Nil(t, err)

	newObj := &models.IntellectualObject{}
	err = json.Unmarshal(jsonBytes, newObj)
	require.Nil(t, err)

	assert.Equal(t, obj.Identifier, newObj.Identifier)
	assert.Equal(t, 20, len(newObj.GenericFiles))
	for _, gf := range newObj.GenericFiles {
		assert.NotEmpty(t, gf.Identifier)
		assert.Equal(t, 2, len(gf.PremisEvents))
		assert.Equal(t, 2, len(gf.Checksums))
	}
}
