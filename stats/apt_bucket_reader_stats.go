package stats

import (
	"github.com/APTrust/exchange/models"
	"github.com/APTrust/exchange/util"
)

type APTBucketReaderStats struct {
	InstitutionsCached        []*models.Institution
	WorkItemsCached           []*models.WorkItem
	WorkItemsFetched          []*models.WorkItem
	WorkItemsCreated          []*models.WorkItem
	WorkItemsQueued           []int
	WorkItemsMarkedAsQueued   []int
	S3Items                   []string
	Errors                    []string
	Warnings                  []string
}

func NewAPTBucketReaderStats() (*APTBucketReaderStats) {
	return &APTBucketReaderStats{
		InstitutionsCached: make([]*models.Institution, 0),
		WorkItemsCached: make([]*models.WorkItem, 0),
		WorkItemsFetched: make([]*models.WorkItem, 0),
		WorkItemsCreated: make([]*models.WorkItem, 0),
		WorkItemsQueued: make([]int, 0),
		WorkItemsMarkedAsQueued: make([]int, 0),
		S3Items: make([]string, 0),
		Errors: make([]string, 0),
		Warnings: make([]string, 0),
	}
}

// Adds an institution to the list of cached institutions.
func (stats *APTBucketReaderStats) AddToInstitutionsCached (inst *models.Institution) {
	stats.InstitutionsCached = append(stats.InstitutionsCached, inst)
}

// Returns true if the Institution with the specified identifier is in
// the Institutions cache.
func (stats *APTBucketReaderStats) InstitutionsCachedContains (identifier string) (bool) {
	return stats.InstitutionByIdentifier(identifier) != nil
}

// Finds an Institution in the cache by identifier. Returns nil if not found.
func (stats *APTBucketReaderStats) InstitutionByIdentifier (identifier string) (*models.Institution) {
	var matchingInst *models.Institution
	for _, inst := range stats.InstitutionsCached {
		if inst.Identifier == identifier {
			matchingInst = inst
			break
		}
	}
	return matchingInst
}

// Adds a WorkItem to the WorkItems cache.
func (stats *APTBucketReaderStats) AddToWorkItemsCached (item *models.WorkItem) {
	stats.WorkItemsCached = append(stats.WorkItemsCached, item)
}

// Returns the item from the WorkItemsCache with the matching name and etag,
// or nil.
func (stats *APTBucketReaderStats) WorkItemsCacheFindByNameAndEtag (name, etag string) (*models.WorkItem) {
	return stats.findWorkItemByNameAndEtag(stats.WorkItemsCached, name, etag)
}

// Returns the item from the WorkItemsCache with the matching id, or nil.
func (stats *APTBucketReaderStats) WorkItemsCacheFindById (id int) (*models.WorkItem) {
	return stats.findWorkItemById(stats.WorkItemsCached, id)
}

// Adds a WorkItem to the list of WorkItems fetched individually from Pharos.
// Items in this list were fetch one at a time because they were not in the
// initial cache.
func (stats *APTBucketReaderStats) AddToWorkItemsFetched (item *models.WorkItem) {
	stats.WorkItemsFetched = append(stats.WorkItemsFetched, item)
}

// Returns the item from WorkItemsFetched with the matching name and etag,
// or nil.
func (stats *APTBucketReaderStats) WorkItemsFetchedFindByNameAndEtag (name, etag string) (*models.WorkItem) {
	return stats.findWorkItemByNameAndEtag(stats.WorkItemsFetched, name, etag)
}

// Returns the item from WorkItemsFetched with the matching id, or nil.
func (stats *APTBucketReaderStats) WorkItemsFetchedFindById (id int) (*models.WorkItem) {
	return stats.findWorkItemById(stats.WorkItemsFetched, id)
}

// Adds a WorkItem to the list WorkItems created by the bucket reader.
func (stats *APTBucketReaderStats) AddToWorkItemsCreated (item *models.WorkItem) {
	stats.WorkItemsCreated = append(stats.WorkItemsCreated, item)
}

// Returns the item from WorkItemsCreated with the matching name and etag,
// or nil.
func (stats *APTBucketReaderStats) WorkItemsCreatedFindByNameAndEtag (name, etag string) (*models.WorkItem) {
	return stats.findWorkItemByNameAndEtag(stats.WorkItemsCreated, name, etag)
}

// Returns the item from WorkItemsCreated with the matching id, or nil.
func (stats *APTBucketReaderStats) WorkItemsCreatedFindById (id int) (*models.WorkItem) {
	return stats.findWorkItemById(stats.WorkItemsCreated, id)
}

// Adds an ID to the list of WorkItem IDs that the bucket reader
// pushed into NSQ.
func (stats *APTBucketReaderStats) AddToWorkItemsQueued (itemId int) {
	stats.WorkItemsQueued = append(stats.WorkItemsQueued, itemId)
}

// Returns true if the work item with the specified ID was queued.
func (stats *APTBucketReaderStats) WorkItemWasQueued (itemId int) (bool) {
	return util.IntListContains(stats.WorkItemsQueued, itemId)
}

// Adds an ID to the list of WorkItems that the bucket reader marked as queued.
func (stats *APTBucketReaderStats) AddToWorkItemsMarkedAsQueued (itemId int) {
	stats.WorkItemsMarkedAsQueued = append(stats.WorkItemsMarkedAsQueued, itemId)
}

// Returns true if the WorkItem with the specified ID was marked as queued.
func (stats *APTBucketReaderStats) WorkItemWasMarkedAsQueued (itemId int) (bool) {
	return util.IntListContains(stats.WorkItemsMarkedAsQueued, itemId)
}

// Adds an item to the list of files that the bucket reader found in the S3
// receiving buckets. Param bucketAndKey should be something like
// "aptrust.receiving.virginia.edu/virginia.edu_12345678.tar"
func (stats *APTBucketReaderStats) AddS3Item (bucketAndKey string) {
	stats.S3Items = append(stats.S3Items, bucketAndKey)
}

// Returns true if the specified bucketAndKey was found in S3
func (stats *APTBucketReaderStats) S3ItemWasFound (bucketAndKey string) (bool) {
	return util.StringListContains(stats.S3Items, bucketAndKey)
}

// Adds an error message to the stats.
func (stats *APTBucketReaderStats) AddError (message string) {
	stats.Errors = append(stats.Errors, message)
}

// Returns true if this object contains any errors
func (stats *APTBucketReaderStats) HasErrors (message string) (bool) {
	return len(stats.Errors) > 0
}

// Adds a warning to the stats.
func (stats *APTBucketReaderStats) AddWarning (message string) {
	stats.Warnings = append(stats.Warnings, message)
}

// Returns true if this object contains any warnings
func (stats *APTBucketReaderStats) HasWarnings (message string) (bool) {
	return len(stats.Warnings) > 0
}

// Returns the WorkItem with the matching name and etag, or nil.
func (stats *APTBucketReaderStats) findWorkItemByNameAndEtag (workItemList []*models.WorkItem, name, etag string) (*models.WorkItem) {
	for _, item := range workItemList {
		if item.Name == name && item.ETag == etag {
			return item
		}
	}
	return nil
}

// Returns the WorkItem with the matching ID, or nil.
func (stats *APTBucketReaderStats) findWorkItemById (workItemList []*models.WorkItem, id int) (*models.WorkItem) {
	for _, item := range workItemList {
		if item.Id == id {
			return item
		}
	}
	return nil
}
