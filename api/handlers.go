package api

import (
	"github.com/gin-gonic/gin"
	d "health/discipline"
	"log"
	"net/http"
)

var (
	h, _ = d.InitHealth()
	q, _ = d.Init()
)

func GetRouter() *gin.Engine {

	router := gin.Default()

	api := router.Group("/api/v1")
	{
		api.POST("/triggers/add", LogTriggerQueue)
		api.POST("/actions/add", LogActionQueue)
		api.POST("/overall/add", LogOverallQueue)
		api.GET("/sink", Sink)
		api.GET("/ping", PingResponse)
	}

	return router
}

func PingResponse(c *gin.Context) {
	c.JSON(200, "test")
}

func LogTriggerQueue(c *gin.Context) {
	// Bind JSON data to trigger struct
	var t d.Trigger
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return // Important: return after sending error response
	}

	// Validate required fields if needed
	if t.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Trigger type is required",
		})
		return
	}

	// Handle LogTrigger error
	if err := q.LogTrigger(t.Type, t.Intensity, t.Compulsion, t.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to log trigger",
			"details": err.Error(),
		})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Trigger logged successfully",
	})
}

func LogActionQueue(c *gin.Context) {
	// Bind JSON data to trigger struct
	var a d.Action
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return // Important: return after sending error response
	}

	// Validate required fields if needed
	if a.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Action type is required",
		})
		return
	}

	// Handle LogTrigger error
	if err := q.LogAction(a.Type, a.Relief, a.Duration, a.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to log Action",
			"details": err.Error(),
		})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Action logged successfully",
	})
}

func LogOverallQueue(c *gin.Context) {
	// Bind JSON data to trigger struct
	var o d.Overall
	if err := c.ShouldBindJSON(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return // Important: return after sending error response
	}

	if err := q.LogOverall(o.CleanStreak, o.GymSession, o.CodingHours, o.ReadingHours, o.MoodScore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to log Overall",
			"details": err.Error(),
		})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Overall logged successfully",
	})
}

func Sink(c *gin.Context) {
	if err := d.RefreshSink(); err != nil {
		log.Println(err)
	}

	c.JSON(http.StatusOK, "Refreshed database")
}
