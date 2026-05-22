package contract

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ygpkg/yg-go/apis/apiobj"
)

// Project 项目响应结构
type Project struct {
	PublicID    string                 `json:"public_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	OwnerID     uint                   `json:"owner_id"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	OwnerID     *uint                   `json:"owner_id,omitempty"`
	Status      *string                 `json:"status,omitempty"`
	Metadata    *map[string]interface{} `json:"metadata,omitempty"`
}

// ListProjectQuery 查询项目列表请求，基于 apiobj.PageQuery
type ListProjectQuery struct {
	apiobj.PageQuery
}

// AllowFilterFields 实现 apiobj.allowFilterFielder 接口
func (ListProjectQuery) AllowFilterFields() []string {
	return []string{"name", "status", "public_id"}
}

// AllowOrderFields 实现 apiobj.allowOrderFielder 接口
func (ListProjectQuery) AllowOrderFields() []string {
	return []string{"created_at", "updated_at", "name"}
}

// Fill 设置分页默认值
func (q *ListProjectQuery) Fill(req *http.Request) {
	q.PageQuery.Fill(req)
	if err := q.PageQuery.IsValite(q); err != nil {
		// 校验错误在 handler 层处理
	}
}

// Validate 校验查询参数
func (q *ListProjectQuery) Validate() error {
	if err := q.PageQuery.IsValite(q); err != nil {
		return err
	}
	if q.Offset <= 0 {
		q.Offset = 0
	}
	return nil
}

// ProjectList 项目列表响应
type ProjectList struct {
	Total  int64     `json:"total"`
	Offset int       `json:"offset"`
	Limit  int       `json:"limit"`
	Items  []Project `json:"items"`
}

// Validate 校验项目名称
func (p *Project) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}
	if p.PublicID == "" {
		return fmt.Errorf("project public_id is required")
	}
	return nil
}
