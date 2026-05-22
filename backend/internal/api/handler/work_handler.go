package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/insmtx/Leros/backend/internal/api/contract"
	"github.com/insmtx/Leros/backend/internal/api/dto"
)

type WorkHandler struct {
	service contract.WorkService
}

func NewWorkHandler(service contract.WorkService) *WorkHandler {
	return &WorkHandler{service: service}
}

func (h *WorkHandler) RegisterRoutes(r gin.IRouter) {
	r.POST("/NewMessage", h.NewMessage)
}

func RegisterWorkRoutes(r gin.IRouter, service contract.WorkService) {
	h := NewWorkHandler(service)
	h.RegisterRoutes(r)
}

// @Summary 首页新建消息
// @Description 原子创建 Project + Task + Session 并分配 AgentWorker
// @Tags Work
// @Accept json
// @Produce json
// @Param body body contract.NewMessageRequest true "新建消息请求"
// @Success 200 {object} dto.BaseResponse "成功响应"
// @Failure 400 {object} dto.ErrorResponse "请求参数错误"
// @Failure 401 {object} dto.ErrorResponse "未认证"
// @Failure 404 {object} dto.ErrorResponse "资源不存在"
// @Failure 500 {object} dto.ErrorResponse "内部服务器错误"
// @Router /NewMessage [post]
func (h *WorkHandler) NewMessage(ctx *gin.Context) {
	var req contract.NewMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.Error(dto.CodeInvalidParams, err.Error()))
		return
	}

	result, err := h.service.NewMessage(ctx, &req)
	if err != nil {
		handleWorkServiceError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, dto.Success(result))
}

func handleWorkServiceError(ctx *gin.Context, err error) {
	if err.Error() == "user not authenticated or org not set" {
		ctx.JSON(http.StatusUnauthorized, dto.Error(dto.CodeInternalError, err.Error()))
		return
	}
	if err.Error() == "permission denied" {
		ctx.JSON(http.StatusForbidden, dto.Error(dto.CodeInternalError, err.Error()))
		return
	}
	if err.Error() == "project not found" || err.Error() == "task not found" {
		ctx.JSON(http.StatusNotFound, dto.Error(dto.CodeNotFound, err.Error()))
		return
	}
	ctx.JSON(http.StatusInternalServerError, dto.Error(dto.CodeInternalError, err.Error()))
}
