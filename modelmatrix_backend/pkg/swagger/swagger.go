package swagger

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Import generated docs
	_ "modelmatrix_backend/docs"
)

// Setup configures Swagger documentation endpoints
func Setup(router *gin.Engine) {
	// Swagger endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
