package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/korgx9/safar-backend/internal/models"
)

type VehicleHandler struct {
	db *gorm.DB
}

func NewVehicleHandler(db *gorm.DB) *VehicleHandler {
	return &VehicleHandler{db: db}
}

// Create godoc
// @Summary Create vehicle
// @Description Creates a vehicle for the authenticated user.
// @Tags Vehicles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateVehicleRequest true "Create vehicle request"
// @Success 201 {object} models.Vehicle
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /vehicles [post]
func (h *VehicleHandler) Create(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	var req models.CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vehicle := models.Vehicle{
		UserID:     userID,
		SeatsTotal: req.SeatsTotal,
	}

	if err := h.db.Create(&vehicle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create vehicle"})
		return
	}

	c.JSON(http.StatusCreated, vehicle)
}

// List godoc
// @Summary List my vehicles
// @Description Returns vehicles owned by the authenticated user.
// @Tags Vehicles
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Vehicle
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /vehicles [get]
func (h *VehicleHandler) List(c *gin.Context) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
		return
	}

	var vehicles []models.Vehicle
	if err := h.db.Where("user_id = ?", userID).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vehicles"})
		return
	}

	c.JSON(http.StatusOK, vehicles)
}
