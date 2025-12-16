package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

// setLinkPreference sets the link preference (ME or Host) on a device's WiFi port.
func (r *deviceManagementRoutes) setLinkPreference(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.LinkPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)

		return
	}

	response, err := r.d.SetLinkPreference(c.Request.Context(), guid, req)
	if err != nil {
		r.l.Error(err, "http - v1 - setLinkPreference")
		// Map specific errors to HTTP status codes (matching MPS implementation)
		if errors.Is(err, wsman.ErrNoWiFiPort) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ErrorResponse(c, err)

		return
	}

	// Map AMT return value to HTTP status code (matching MPS implementation)
	// MPS logic: null -> 404, -1 or non-zero -> 400, 0 -> 200
	httpStatus := http.StatusOK
	if response.ReturnValue == -1 || response.ReturnValue != 0 {
		httpStatus = http.StatusBadRequest
	}

	c.JSON(httpStatus, response)
}
