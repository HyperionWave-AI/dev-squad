package handlers

import (
	"hyper/internal/mcp/storage"

	"github.com/stretchr/testify/mock"
)

// CompleteKnowledgeStorageMock is a comprehensive mock implementation with testify/mock
type CompleteKnowledgeStorageMock struct {
	mock.Mock
}

func (m *CompleteKnowledgeStorageMock) Upsert(collection, text string, metadata map[string]interface{}, taskId *string) (*storage.KnowledgeEntry, error) {
	args := m.Called(collection, text, metadata, taskId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.KnowledgeEntry), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) UpdateEntry(id, text string, metadata map[string]interface{}) (*storage.KnowledgeEntry, error) {
	args := m.Called(id, text, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.KnowledgeEntry), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) DeleteEntry(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *CompleteKnowledgeStorageMock) GetEntryByID(id string) (*storage.KnowledgeEntry, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.KnowledgeEntry), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) GetEntriesByCollection(collectionName string) ([]*storage.KnowledgeEntry, error) {
	args := m.Called(collectionName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.KnowledgeEntry), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) Query(collection, query string, limit int, taskId *string, voteBoost ...float64) ([]*storage.QueryResult, error) {
	args := m.Called(collection, query, limit, taskId, voteBoost)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.QueryResult), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) ListCollections() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *CompleteKnowledgeStorageMock) CreateCollection(name, category, description string, tags []string) (*storage.Collection, error) {
	args := m.Called(name, category, description, tags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Collection), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) DeleteCollection(id string) (string, int64, error) {
	args := m.Called(id)
	return args.String(0), args.Get(1).(int64), args.Error(2)
}

func (m *CompleteKnowledgeStorageMock) GetPopularCollections(limit int) ([]*storage.CollectionStats, error) {
	args := m.Called(limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.CollectionStats), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) GetCollectionStatsWithMetadata() ([]*storage.CollectionWithMetadata, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.CollectionWithMetadata), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) ListKnowledge(collection string, limit int) ([]*storage.KnowledgeEntry, error) {
	args := m.Called(collection, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.KnowledgeEntry), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) UpdateCollectionMetadata(collectionName, description string, tags []string, category string) (*storage.CollectionMetadata, error) {
	args := m.Called(collectionName, description, tags, category)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.CollectionMetadata), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) RenameCollection(oldName, newName string) (int64, error) {
	args := m.Called(oldName, newName)
	return args.Get(0).(int64), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) VoteOnEntry(entryID, userID, vote, reason string) (*storage.Vote, error) {
	args := m.Called(entryID, userID, vote, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.Vote), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) GetEntryVotes(entryID, userID string) (*storage.VoteSummary, error) {
	args := m.Called(entryID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.VoteSummary), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) BatchSyncVotesToQdrant(collectionName string) (int, error) {
	args := m.Called(collectionName)
	return args.Int(0), args.Error(1)
}

func (m *CompleteKnowledgeStorageMock) ExportToFiles(outputPath string, collections []string) (*storage.ExportReport, error) {
	args := m.Called(outputPath, collections)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ExportReport), args.Error(1)
}
