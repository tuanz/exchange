package tarfile

import (
	"archive/tar"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"github.com/APTrust/exchange/models"
	"github.com/APTrust/exchange/platform"
	"github.com/APTrust/exchange/util"
	"github.com/satori/go.uuid"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Reader struct {
	Manifest     *models.IngestManifest
	tarReader    *tar.Reader
}

func NewReader(manifest *models.IngestManifest) (*Reader) {
	return &Reader{
		Manifest: manifest,
	}
}

// absInputFile -> reader.Manifest.Object.IngestTarFilePath
// bagName -> reader.Manifest.Object.BagName
func (reader *Reader) Untar() {
	reader.RecordStartOfWork()
	if !reader.ManifestInfoIsValid() {
		reader.Manifest.Untar.Finish()
		return
	}

	// Note the tar file's parent directory
	tarFileDir := filepath.Dir(reader.Manifest.Object.IngestUntarredPath)

	// Open the tar file for reading.
	file, err := os.Open(reader.Manifest.Object.IngestTarFilePath)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		reader.Manifest.Untar.AddError("Could not open file %s for untarring: %v", reader.Manifest.Object.IngestTarFilePath, err)
		reader.Manifest.Untar.Finish()
		return
	}

	// Untar the file and record the results.
	reader.tarReader = tar.NewReader(file)

	for {
		header, err := reader.tarReader.Next()
		if err != nil && err.Error() == "EOF" {
			break // end of archive
		}
		if err != nil {
			reader.Manifest.Untar.AddError("Error reading tar file header: %v. " +
				"Either this is not a tar file, or the file is corrupt.", err)
			reader.Manifest.Untar.Finish()
			return
		}

		// Top-level dir will be the first header entry.
		if header.Typeflag == tar.TypeDir && reader.Manifest.Object.IngestUntarredPath == "" {
			topLevelDir, err := reader.GetTopLevelDir(header.Name)
			if err != nil {
				reader.Manifest.Untar.AddError(err.Error())
				reader.Manifest.Untar.Finish()
				return
			}
			reader.Manifest.Object.IngestUntarredPath = filepath.Join(tarFileDir, topLevelDir)
		}

		// Get the output path for this file -> Where should we untar it to?
		outputPath := filepath.Join(reader.Manifest.Object.IngestUntarredPath, header.Name)

		// Make sure the directory that we're about to write into exists.
		err = os.MkdirAll(filepath.Dir(outputPath), 0755)
		if err != nil {
			reader.Manifest.Untar.AddError("Could not create destination file '%s' "+
				"while unpacking tar archive: %v", outputPath, err)
			return
		}

		// Copy the file, if it's an actual file. Otherwise, ignore it and record
		// a warning. The bag library does not deal with items like symlinks.
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			fileName, err := GetFileName(header.Name)
			if err != nil {
				reader.Manifest.Untar.AddError(err.Error())
				reader.Manifest.Untar.Finish()
				return
			}
			if util.HasSavableName(fileName) {
				gf := reader.CreateAndSaveGenericFile(fileName, header)
				if gf.IngestErrorMessage != "" {
					reader.Manifest.Untar.AddError(gf.IngestErrorMessage)
					reader.Manifest.Untar.Finish()
					return
				}
			} else {
				// This is probably something like bagit.txt or a manifest,
				// which we must save to disk but won't need to preserve in
				// long-term storage
				err = reader.SaveFile(outputPath)
				if err != nil {
					reader.Manifest.Untar.AddError(
						"Error copying file from tar archive to '%s': %v",
						outputPath, err)
					reader.Manifest.Untar.Finish()
					return
				}
			}
		} else if header.Typeflag != tar.TypeDir {
			// Header item is neither file nor directory.
			// Do nothing, but record that we saw this item.
			reader.Manifest.Object.IngestFilesIgnored = append(
				reader.Manifest.Object.IngestFilesIgnored,
				header.Name)
		}
	}
	reader.Manifest.Untar.Finish()
}

// Record that we're starting on this.
func (reader *Reader) RecordStartOfWork() {
	reader.Manifest.Untar.Attempted = true
	reader.Manifest.Untar.AttemptNumber += 1
	reader.Manifest.Untar.FinishedAt = time.Time{}
	reader.Manifest.Untar.Start()
}

// Make sure the manifest has enough information
// for us to get started.
func (reader *Reader) ManifestInfoIsValid() (bool) {
	if reader.Manifest.Object == nil {
		reader.Manifest.Untar.AddError("IntellectualObject is missing from manifest.")
		return false
	}
	if reader.Manifest.Object.Identifier == "" {
		reader.Manifest.Untar.AddError("IntellectualObject has no Identifier.")
	}
	if reader.Manifest.Object.BagName == "" {
		reader.Manifest.Untar.AddError("IntellectualObject has no BagName.")
	}
	if reader.Manifest.Object.Institution == "" {
		reader.Manifest.Untar.AddError("IntellectualObject has no Institution.")
	}
	tarFilePath := reader.Manifest.Object.IngestTarFilePath
	if tarFilePath == "" {
		reader.Manifest.Untar.AddError("IntellectualObject is missing IngestTarFilePath.")
	} else if absPath, _ := filepath.Abs(tarFilePath); absPath != tarFilePath {
		reader.Manifest.Untar.AddError("IntellectualObject has a relative or incorrect IngestTarFilePath.")
	}
	if fileStat, err := os.Stat(tarFilePath); os.IsNotExist(err) {
		reader.Manifest.Untar.AddError("IngestTarFilePath '%s' does not exist.", tarFilePath)
	} else if fileStat.Mode().IsDir() {
		reader.Manifest.Untar.AddError("IngestTarFilePath '%s' is a directory.", tarFilePath)
	}
	return reader.Manifest.Untar.HasErrors()
}

// Saves the file to disk and returns a GenericFile object.
func (reader *Reader) CreateAndSaveGenericFile(fileName string, header *tar.Header) (*models.GenericFile) {
	fileDir := filepath.Dir(reader.Manifest.Object.IngestUntarredPath)
	gf := models.NewGenericFile()
	reader.Manifest.Object.GenericFiles = append(reader.Manifest.Object.GenericFiles, gf)
	var err error
	gf.IngestLocalPath, err = filepath.Abs(filepath.Join(fileDir, header.Name))
	if err != nil {
		gf.IngestErrorMessage = fmt.Sprintf("Path error: %v", err)
		reader.Manifest.Untar.AddError(gf.IngestErrorMessage)
		return gf
	}
	gf.IntellectualObjectIdentifier = reader.Manifest.Object.Identifier
	gf.Identifier = fmt.Sprintf("%s/%s", reader.Manifest.Object.Identifier, gf.IngestLocalPath)
	gf.FileModified = header.ModTime
	gf.Size = header.Size
	gf.IngestFileUid = header.Uid
	gf.IngestFileGid = header.Gid
	gf.IngestFileUname = header.Uname
	gf.IngestFileGname = header.Gname
	gf.IngestUUID = uuid.NewV4().String()
	gf.IngestUUIDGeneratedAt = time.Now().UTC()
	reader.SaveWithChecksums(gf)
	return gf
}

// Saves a file from the tar archive to local disk. This function
// used to save non-data files (manifests, tag files, etc.)
func (reader *Reader) SaveFile(destination string) error {
	// TODO: Save with same permissions as file in tar archive
	outputWriter, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY, 0644)
	if outputWriter != nil {
		defer outputWriter.Close()
	}
	if err != nil {
		return err
	}
	_, err = io.Copy(outputWriter, reader.tarReader)
	if err != nil {
		return err
	}
	return nil
}

// Returns the relative path of the top-level directory into which a
// tar file expands. According to APTrust specs, my_bag.tar should
// expand into a directory called my_bag (the dir name should match
// the tar file name, minus the .tar extension). This isn't always the
// case with bags we get from depositors. So this figures out what that
// top-level directory actually is, and lets us know if there's an error.
func (reader *Reader)GetTopLevelDir(headerName string) (topLevelDir string, err error) {
	topLevelDir = strings.Replace(headerName, "/", "", 1)
	// Fix for Windows
	systemNormalizedPath := topLevelDir
	if runtime.GOOS == "windows" && strings.Contains(topLevelDir, "\\") {
		systemNormalizedPath = strings.Replace(topLevelDir, "\\", "/", -1)
	}
	expectedDir := path.Base(systemNormalizedPath)
	if strings.HasSuffix(expectedDir, ".tar") {
		expectedDir = expectedDir[0 : len(expectedDir)-4]
	}
	if topLevelDir != expectedDir {
		err = fmt.Errorf("Bag '%s' should untar to a folder named '%s', but "+
			"it untars to '%s'. Please repackage this bag and try again.",
			path.Base(reader.Manifest.Object.IngestTarFilePath), expectedDir, topLevelDir)
	}
	return topLevelDir, err
}

func GetFileName(headerName string) (string, error) {
	pathParts := strings.SplitN(headerName, "/", 2)
	if len(pathParts) < 2 {
		err := fmt.Errorf("File %s in tar archive should be in format dir/filename", headerName)
		return "", err
	}
	return pathParts[1], nil
}

// buildFile saves a data file from the tar archive to disk,
// then returns a struct with data we'll need to construct the
// GenericFile object in Fedora later.
func (reader *Reader)SaveWithChecksums(gf *models.GenericFile) {
	// Set up a MultiWriter to stream data ONCE to file,
	// md5 and sha256. We don't want to process the stream
	// three separate times.
	outputWriter, err := os.OpenFile(gf.IngestLocalPath, os.O_CREATE|os.O_WRONLY, 0644)
	if outputWriter != nil {
		defer outputWriter.Close()
	}
	if err != nil {
		gf.IngestErrorMessage = fmt.Sprintf("Error opening writing to %s: %v", gf.IngestLocalPath, err)
		return
	}
	md5Hash := md5.New()
	shaHash := sha256.New()
	multiWriter := io.MultiWriter(md5Hash, shaHash, outputWriter)
	io.Copy(multiWriter, reader.tarReader)
	gf.IngestMd5 = fmt.Sprintf("%x", md5Hash.Sum(nil))
	gf.IngestSha256 = fmt.Sprintf("%x", shaHash.Sum(nil))
	gf.IngestSha256GeneratedAt = time.Now().UTC()
	gf.FileFormat, _ = platform.GuessMimeType(gf.IngestLocalPath)  // on err, defaults to application/binary
	return
}

// Adds a file to a tar archive.
func AddToArchive(tarWriter *tar.Writer, filePath, pathWithinArchive string) (error) {
	finfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("Cannot add '%s' to archive: %v", filePath, err)
	}
	header := &tar.Header{
		Name: pathWithinArchive,
		Size: finfo.Size(),
		Mode: int64(finfo.Mode().Perm()),
		ModTime: finfo.ModTime(),
	}

	// This call adds the owner and group info to the tar file header.
	// When running on *nix systems that support this call, we use
	// the definition in nix.go. On Windows, which does not support
	// the call, we use the no-op definition in windows.go.
	platform.GetOwnerAndGroup(finfo, header)

	// Write the header entry
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	// Open the file whose data we're going to add.
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		return err
	}

	// Copy the contents of the file into the tarWriter.
	bytesWritten, err := io.Copy(tarWriter, file)
	if bytesWritten != header.Size {
		return fmt.Errorf("addToArchive() copied only %d of %d bytes for file %s",
			bytesWritten, header.Size, filePath)
	}
	if err != nil {
		return fmt.Errorf("Error copying %s into tar archive: %v",
			filePath, err)
	}

	return nil
}
