package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
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
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}
