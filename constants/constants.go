// Common vars and constants, shared by many parts of the bagman library.
package constants

import (
	"regexp"
)


// The tar files that make up multipart bags include a suffix
// that follows this pattern. For example, after stripping off
// the .tar suffix, you'll have a name like "my_bag.b04.of12"
var MultipartSuffix = regexp.MustCompile("\\.b\\d+\\.of\\d+$")

// Regex for a valid APTrust file name, according to the spec at
// https://sites.google.com/a/aptrust.org/member-wiki/basic-operations/bagging
// This regex says a valid file name can be exactly one alpha-numeric character,
// or 2+ characters, beginning with alpha-numerics or dot or underscore,
// followed by alphanumerics, dots, underscores, dashes and percent signs.
var APTrustFileNamePattern = regexp.MustCompile("^([A-Za-z0-9])$|^([A-Za-z0-9\\._][A-Za-z0-9\\.\\-_%]+)$")

// Regex for valid POSIX filenames.
var PosixFileNamePattern = regexp.MustCompile("^[A-Za-z0-9\\._\\-]+$")

const APTrustSystemUser = "system@aptrust.org"

const (
	APTrustNamespace        = "urn:mace:aptrust.org"
	ReceiveBucketPrefix     = "aptrust.receiving."
	ReceiveTestBucketPrefix = "aptrust.receiving.test."
	RestoreBucketPrefix     = "aptrust.restore."
	S3DateFormat            = "2006-01-02T15:04:05.000Z"
	// All S3 urls begin with this.
	S3UriPrefix             = "https://s3.amazonaws.com/"
)


// Status enumerations match values defined in
// https://github.com/APTrust/fluctus/blob/develop/config/application.rb
const (
	StatusStarted              = "Started"
	StatusPending              = "Pending"
	StatusSuccess              = "Success"
	StatusFailed               = "Failed"
	StatusCancelled            = "Cancelled"
)

var StatusTypes []string = []string{
	StatusStarted,
	StatusPending,
	StatusSuccess,
	StatusFailed,
	StatusCancelled,
}

// Stage enumerations match values defined in
// https://github.com/APTrust/fluctus/blob/develop/config/application.rb
const (
	StageRequested           = "Requested"
	StageReceive             = "Receive"
	StageFetch               = "Fetch"
	StageUnpack              = "Unpack"
	StageValidate            = "Validate"
	StageStore               = "Store"
	StageRecord              = "Record"
	StageCleanup             = "Cleanup"
	StageResolve             = "Resolve"
)

var StageTypes []string = []string{
	StageRequested,
	StageReceive,
	StageFetch,
	StageUnpack,
	StageValidate,
	StageStore,
	StageRecord,
	StageCleanup,
	StageResolve,
}

// Action enumerations match values defined in
// https://github.com/APTrust/fluctus/blob/develop/config/application.rb

const (
	ActionIngest                 = "Ingest"
	ActionFixityCheck            = "Fixity Check"
	ActionRestore                = "Restore"
	ActionDelete                 = "Delete"
	ActionDPN                    = "DPN"
)

var ActionTypes []string = []string{
	ActionIngest,
	ActionFixityCheck,
	ActionRestore,
	ActionDelete,
}


const (
	AlgMd5                      = "md5"
	AlgSha256                   = "sha256"
)

var ChecksumAlgorithms = []string{ AlgMd5, AlgSha256 }

const (
	IdTypeStorageURL                 = "url"
	IdTypeBagAndPath                 = "uuid"
)

// List of valid APTrust IntellectualObject AccessRights.
var AccessRights []string = []string{
	"consortia",
	"institution",
	"restricted",
}

// AWS Regions (the ones we're using for storage)
const (
	AWSVirginia = "us-east-1"
	AWSOregon = "us-west-2"
)

// GenericFile types. GenericFile.IngestFileType
const (
	PAYLOAD_FILE     = "payload_file"
	PAYLOAD_MANIFEST = "payload_manifest"
	TAG_MANIFEST     = "tag_manifest"
	TAG_FILE         = "tag_file"
)

// PREMIS Event types as defined by the Library of Congress at
// http://id.loc.gov/search/?q=&q=cs%3Ahttp%3A%2F%2Fid.loc.gov%2Fvocabulary%2Fpreservation%2FeventType#
const(
	// The process whereby a repository actively obtains an object.
	EventCapture = "capture"

	// The process of coding data to save storage space or transmission time.
	EventCompression = "compression"

	// The act of creating a new object.
	EventCreation = "creation"

	// The process of removing an object from the inventory of a repository.
	EventDeaccession = "deaccession"

	// The process of reversing the effects of compression.
	EventDecompression = "decompression"

	//The process of converting encrypted data to plain text.
	EventDecryption = "decryption"

	// The process of removing an object from repository storage.
	EventDeletion = "deletion"

	// The process by which a message digest ("hash") is created.
	EventDigestCalculation = "message digest calculation"

	// The process of verifying that an object has not been changed in a given period.
	EventFixityCheck = "fixity check"

	// The process of adding objects to a preservation repository.
	EventIngestion = "ingestion"

	// A transformation of an object creating a version in a more contemporary format.
	EventMigration = "migration"

	// A transformation of an object creating a version more conducive to preservation.
	EventNormalization = "normalization"

	// The process of creating a copy of an object that is, bit-wise, identical to the original.
	EventReplication = "replication"

	// The process of determining that a decrypted digital signature matches an expected value.
	EventSignatureValidation = "digital signature validation"

	// The process of comparing an object with a standard and noting compliance or exceptions.
	EventValidation = "validation"

	// The process of scanning a file for malicious programs.
	EventVirusCheck = "virus check"
)

var EventTypes []string = []string{
	EventCapture,
	EventCompression,
	EventCreation,
	EventDeaccession,
	EventDecompression,
	EventDecryption,
	EventDeletion,
	EventDigestCalculation,
	EventFixityCheck,
	EventIngestion,
	EventMigration,
	EventNormalization,
	EventReplication,
	EventSignatureValidation,
	EventValidation,
	EventVirusCheck,
}
