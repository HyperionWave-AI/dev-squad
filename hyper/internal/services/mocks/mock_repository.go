package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockCollection is a mock implementation of mongo.Collection
type MockCollection struct {
	mock.Mock
}

// NewMockCollection creates a new mock collection
func NewMockCollection() *MockCollection {
	return &MockCollection{}
}

// InsertOne mocks the InsertOne method
func (m *MockCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

// Find mocks the Find method
func (m *MockCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

// FindOne mocks the FindOne method
func (m *MockCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return &mongo.SingleResult{}
	}
	return args.Get(0).(*mongo.SingleResult)
}

// UpdateOne mocks the UpdateOne method
func (m *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

// UpdateMany mocks the UpdateMany method
func (m *MockCollection) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

// DeleteOne mocks the DeleteOne method
func (m *MockCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

// DeleteMany mocks the DeleteMany method
func (m *MockCollection) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

// CountDocuments mocks the CountDocuments method
func (m *MockCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(int64), args.Error(1)
}

// Indexes mocks the Indexes method
func (m *MockCollection) Indexes() mongo.IndexView {
	args := m.Called()
	if args.Get(0) == nil {
		return mongo.IndexView{}
	}
	return args.Get(0).(mongo.IndexView)
}

// MockSingleResult is a mock implementation of mongo.SingleResult
type MockSingleResult struct {
	mock.Mock
}

// Decode mocks the Decode method
func (m *MockSingleResult) Decode(v interface{}) error {
	args := m.Called(v)
	return args.Error(0)
}

// MockCursor is a mock implementation of mongo.Cursor
type MockCursor struct {
	mock.Mock
}

// Close mocks the Close method
func (m *MockCursor) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Next mocks the Next method
func (m *MockCursor) Next(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

// Decode mocks the Decode method
func (m *MockCursor) Decode(v interface{}) error {
	args := m.Called(v)
	return args.Error(0)
}

// All mocks the All method
func (m *MockCursor) All(ctx context.Context, results interface{}) error {
	args := m.Called(ctx, results)
	return args.Error(0)
}

// MockIndexView is a mock implementation of mongo.IndexView
type MockIndexView struct {
	mock.Mock
}

// CreateOne mocks the CreateOne method
func (m *MockIndexView) CreateOne(ctx context.Context, model mongo.IndexModel, opts ...*options.CreateIndexesOptions) (string, error) {
	args := m.Called(ctx, model, opts)
	return args.String(0), args.Error(1)
}

// MockDatabase is a mock implementation of mongo.Database
type MockDatabase struct {
	mock.Mock
}

// Collection mocks the Collection method
func (m *MockDatabase) Collection(name string, opts ...*options.CollectionOptions) *mongo.Collection {
	args := m.Called(name, opts)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*mongo.Collection)
}

// Client mocks the Client method
func (m *MockDatabase) Client() *mongo.Client {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*mongo.Client)
}