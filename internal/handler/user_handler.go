package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"voca-test/internal/models"
	"voca-test/internal/service"
	"voca-test/internal/util"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{userService: s}
}

// CreateUser - Create a new user
// @Summary Create a new user
// @Description Register a new user in the system with the provided name.
// @Tags Users
// @Accept json
// @Produce json
// @Param user body models.UserCreateRequest true "User registration details"
// @Success 201 {object} util.APIResponse{data=models.User} "User created successfully"
// @Failure 400 {object} util.APIResponse "Invalid request body or validation error"
// @Failure 500 {object} util.APIResponse "Internal server error"
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fieldErrors := util.ParseValidationErrors(err)
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", fieldErrors)
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req.Name)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to create user", err.Error())
		return
	}
	util.SuccessResponse(c, http.StatusCreated, user, "User created successfully")
}
