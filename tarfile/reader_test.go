package tarfile_test

import (
	"fmt"
	"github.com/APTrust/exchange/models"
	"github.com/APTrust/exchange/tarfile"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Return a manifest pointing to our test tar file.
func getReader(tarFileName string) (*tarfile.Reader) {
	_, filename, _, _ := runtime.Caller(1)
	tarFilePath, _ := filepath.Abs(path.Join(filepath.Dir(filename), "..", "testdata", tarFileName))
	objIdentifier := strings.Replace(tarFileName, ".tar", "", 1)
	parts := strings.Split(objIdentifier, ".")
	institution := fmt.Sprintf("%s.%s", parts[0], parts[1])
	bagName := strings.Replace(objIdentifier, institution + ".", "", 1)

	manifest := models.NewIngestManifest()
	manifest.Object.Identifier = objIdentifier
	manifest.Object.Institution = institution
	manifest.Object.BagName = bagName
	manifest.Object.IngestTarFilePath = tarFilePath

	return tarfile.NewReader(manifest)
}

func TestNewReader(t *testing.T) {
	r := getReader("virginia.edu.uva-lib_2278801.tar")
	if r.Manifest == nil {
		assert.FailNow(t, "Reader Manifest should not be nil")
	}
	if r.Manifest.Object == nil {
		assert.FailNow(t, "Reader Manifest Object should not be nil")
	}
	assert.Equal(t, "virginia.edu.uva-lib_2278801", r.Manifest.Object.Identifier)
	assert.Equal(t, "virginia.edu", r.Manifest.Object.Institution)
	assert.Equal(t, "uva-lib_2278801", r.Manifest.Object.BagName)
	assert.True(t, strings.HasPrefix(r.Manifest.Object.IngestTarFilePath, "/"))
	assert.True(t, strings.HasSuffix(r.Manifest.Object.IngestTarFilePath, "testdata/virginia.edu.uva-lib_2278801.tar"))
}

func TestRecordStartOfWork(t *testing.T) {
	r := getReader("example.edu.tagsample_good.tar")
	outputPath := strings.Replace(r.Manifest.Object.IngestTarFilePath, ".tar", "", -1)
	if len(outputPath) > 40 && strings.Contains(outputPath, "testdata") {
		defer os.RemoveAll(outputPath)
	}
	r.Untar()
	assert.True(t, r.Manifest.Untar.Attempted)
	assert.Equal(t, 1, r.Manifest.Untar.AttemptNumber)
	assert.False(t, r.Manifest.Untar.StartedAt.IsZero())
	assert.False(t, r.Manifest.Untar.FinishedAt.IsZero())
}

func TestManifestInfoIsValid(t *testing.T) {
	// Should flag all missing items
	r := getReader("virginia.edu.uva-lib_2278801.tar")
	r.Manifest.Object.Identifier = ""
	r.Manifest.Object.BagName = ""
	r.Manifest.Object.Institution = ""
	r.Manifest.Object.IngestTarFilePath = ""
	r.Untar()
	assert.Equal(t, 5, len(r.Manifest.Untar.Errors))

	// Should be specific about bad file path
	r = getReader("virginia.edu.uva-lib_2278801.tar")
	r.Manifest.Object.IngestTarFilePath = "/mUje9Dke0776adBx4Gq/file/does/not/exist.tar"
	r.Untar()
	if r.Manifest.Untar.HasErrors() == false {
		assert.Fail(t, "Untar WorkSummary should have errors")
	} else {
		assert.True(t, strings.Contains(r.Manifest.Untar.Errors[0], "does not exist"))
	}

	// If IntellectualObject is nil, we should get an
	// error message and not a panic.
	r = getReader("virginia.edu.uva-lib_2278801.tar")
	r.Manifest.Object = nil
	r.Untar()
	if r.Manifest.Untar.HasErrors() == false {
		assert.Fail(t, "Untar WorkSummary should have errors")
	} else {
		assert.Equal(t, "IntellectualObject is missing from manifest.", r.Manifest.Untar.Errors[0])
	}
}

func TestCreateAndSaveGenericFile(t *testing.T) {

}

func TestSaveFile(t *testing.T) {

}

func GetTopLevelDir(t *testing.T) {

}

func TestGetFileName(t *testing.T) {

}

func TestSaveWithChecksums(t *testing.T) {

}

func TestUntar(t *testing.T) {

}
