package models

import "time"

// HTTPAuthType represents authentication type for HTTP tools
type HTTPAuthType string

const (
	AuthTypeNone   HTTPAuthType = "none"
	AuthTypeBearer HTTPAuthType = "bearer"
	AuthTypeAPIKey HTTPAuthType = "apiKey"
	AuthTypeBasic  HTTPAuthType = "basic"
)

// HTTPMethod represents HTTP methods for tools
type HTTPMethod string

const (
	HTTPMethodGET    HTTPMethod = "GET"
	HTTPMethodPOST   HTTPMethod = "POST"
	HTTPMethodPUT    HTTPMethod = "PUT"
	HTTPMethodDELETE HTTPMethod = "DELETE"
	HTTPMethodPATCH  HTTPMethod = "PATCH"
)

// HTTPToolParameter represents a parameter for HTTP tool
type HTTPToolParameter struct {
	Name        string `json:"name" bson:"name"`
	Description string `json:"description" bson:"description"`
	Type        string `json:"type" bson:"type"`                   // string, number, boolean, object, array
	Required    bool   `json:"required" bson:"required"`
	Default     string `json:"default,omitempty" bson:"default,omitempty"`
}

// HTTPToolDefinition represents an HTTP-based tool definition
type HTTPToolDefinition struct {
	ID             string              `json:"id" bson:"_id"`
	ToolName       string              `json:"toolName" bson:"toolName"`
	Description    string              `json:"description" bson:"description"`
	Endpoint       string              `json:"endpoint" bson:"endpoint"`
	Method         HTTPMethod          `json:"method" bson:"method"`
	Headers        map[string]string   `json:"headers,omitempty" bson:"headers,omitempty"`
	Parameters     []HTTPToolParameter `json:"parameters,omitempty" bson:"parameters,omitempty"`
	AuthType       HTTPAuthType        `json:"authType" bson:"authType"`
	AuthTokenField string              `json:"authTokenField,omitempty" bson:"authTokenField,omitempty"` // Field name for auth token in headers
	CompanyID      string              `json:"companyId" bson:"companyId"`
	CreatedBy      string              `json:"createdBy" bson:"createdBy"` // User email
	CreatedAt      time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// CreateHTTPToolRequest represents request to create HTTP tool
type CreateHTTPToolRequest struct {
	ToolName       string              `json:"toolName" binding:"required"`
	Description    string              `json:"description" binding:"required"`
	Endpoint       string              `json:"endpoint" binding:"required"`
	Method         HTTPMethod          `json:"method" binding:"required"`
	Headers        map[string]string   `json:"headers"`
	Parameters     []HTTPToolParameter `json:"parameters"`
	AuthType       HTTPAuthType        `json:"authType" binding:"required"`
	AuthTokenField string              `json:"authTokenField"`
}

// HTTPToolListResponse represents paginated list of HTTP tools
type HTTPToolListResponse struct {
	Tools      []HTTPToolDefinition `json:"tools"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalPages int                  `json:"totalPages"`
}
