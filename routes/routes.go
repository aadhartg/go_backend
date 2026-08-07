// routes/routes.go
package routes

import (
	"theransticslabs/m/controllers"
	"theransticslabs/m/middlewares"
	"theransticslabs/m/utils"

	"time"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	// Apply Logging Middleware
	router.Use(middlewares.LoggingMiddleware())

	// Define Routes
	router.GET(utils.RouteWelcome, controllers.WelcomeHandler)
	router.POST(utils.RouteLogin, middlewares.RateLimit(50, time.Minute), controllers.LoginHandler)
	router.POST(utils.RoutePatientRegister, middlewares.RateLimit(50, time.Minute), controllers.PatientRegister)
	router.POST(utils.RouteForgetPassword, middlewares.RateLimit(50, time.Minute), controllers.ForgetPasswordHandler)
	router.GET(utils.RouteFetchQuantityDiscount, controllers.FetchQuantityDiscountList)
	router.POST(utils.RouteEncryptProductDetails, controllers.EncryptProductDetails)
	router.POST(utils.RouteVerifyProductDetails, controllers.VerifyProduct)
	router.POST(utils.RouteProductPaymentDetails, controllers.OrderCreateHandler)
	router.GET(utils.RoutePaymentSuccessPaypal, controllers.HandlePaymentSuccess)
	router.POST(utils.RouteVerifyBarcode, controllers.VerifyBarcode)
	router.POST(utils.RouteKitRegister, controllers.KitRegister)
	router.POST(utils.RouteEncryptData, controllers.EncryptionDetails)

	consent := router.Group("/api/consent")
	consent.Use(middlewares.ConsentSessionMiddleware())
	consent.Use(middlewares.RateLimit(100, time.Minute))

	// Protected Routes
	protected := router.Group("/api")
	protected.Use(middlewares.AuthMiddleware())
	protected.Use(middlewares.CreatePermissionMiddleware())
	protected.Use(middlewares.RateLimit(500, time.Minute))
	// Add protected routes here
	protected.DELETE(utils.RouteLogout, controllers.LogoutHandler)
	protected.PATCH(utils.RouteResetPassword, controllers.ResetPasswordHandler)

	//patient route
	protected.GET(utils.RouteGetAdminPatientList, controllers.GetAllPatientHandler)
	protected.POST(utils.RouteCreateAdminPatient, controllers.PatientRegister)
	protected.PATCH(utils.RouteUpdateAdminPatientStatus, controllers.UpdateUserStatusHandler)
	protected.PATCH(utils.RouteUpdateAdminPatientDetails, controllers.UpdatePatientDetailsHandler)

	// User routes
	protected.PATCH(utils.RouteUpdateUser, controllers.UpdateUserInfoHandler)
	protected.GET(utils.RouteGetUserProfile, controllers.GetUserProfileHandler)
	protected.GET(utils.RouteGetAdminUserList, controllers.GetAdminUsersHandler)
	protected.POST(utils.RouteCreateAdminUser, controllers.CreateUserHandler)
	protected.PATCH(utils.RouteUpdateAdminUserProfile, controllers.UpdateUserProfileHandler)
	protected.PATCH(utils.RouteUpdateAdminUserStatus, controllers.UpdateUserStatusHandler)
	protected.PATCH(utils.RouteUpdateAdminUserPassword, controllers.UpdateUserPasswordHandler)
	protected.DELETE(utils.RouteDeleteAdminUser, controllers.DeleteUserHandler)

	// Notification routes
	protected.GET(utils.RouteNotification, controllers.GetUserNotifications)
	protected.GET(utils.RouteLatestNotification, controllers.GetLatestNotifications)
	protected.PATCH(utils.RouteMarkNotificationAsRead, controllers.MarkNotificationRead)
	protected.PATCH(utils.RouteMarkAllNotificationsAsRead, controllers.MarkAllNotificationsRead)
	protected.DELETE(utils.RouteNotification, controllers.DeleteUserNotifications)
	protected.DELETE(utils.RouteDeleteNotification, controllers.DeleteNotification)

	// Manage Inventory routes
	protected.POST(utils.RouteKitInfo, controllers.CreateKitHandler)
	protected.GET(utils.RouteKitInfo, controllers.GetKitsListHandler)
	protected.PATCH(utils.RouteKitInfoID, controllers.UpdateKitHandler)
	protected.DELETE(utils.RouteKitInfoID, controllers.DeleteKitHandler)
	protected.GET(utils.RouteQuantitySummary, controllers.GetKitsQuantitySummaryHandler)

	// Manage Customer routes
	protected.GET(utils.RouteviewCustomerWithOrders, controllers.GetCustomerOrderDetails)
	protected.GET(utils.RouteCustomer, controllers.GetCustomersWithOrders)

	// Manage Order routes
	protected.GET(utils.RouteOrder, controllers.GetOrdersWithCustomersInvoicePayments)
	protected.POST(utils.RouteAssignKit, controllers.AssignKit)
	protected.PATCH(utils.RouteChangeOrderStatus, controllers.UpdateOrderStatus)
	protected.GET(utils.RouteOrderCount, controllers.GetOrderCounts)

	// Manage Patient routes
	protected.GET(utils.RouteFetchedKitRegistrationList, controllers.GetPatientRegisterKitList)
	protected.PATCH(utils.RoutePatientUpdateStatus, controllers.UpdatePatientStatus)
	protected.PATCH(utils.RoutePatientUploadReport, controllers.FileUpload)

	// Global Search routes
	protected.GET(utils.RouteFetchedBarcodeDetails, controllers.FetchBarcodeDetails)

	// Manage Labs
	protected.GET(utils.RouteFetchedLabsDetails, controllers.GetAllLabs)

	// Admin dashboard routes
	protected.GET(utils.RouteAdminDashboardStats, controllers.GetDashboardStats)
	protected.GET(utils.RouteAdminDashboardDropDownData, controllers.GetEConsentDropdownData)

	// Video routes
	protected.POST(utils.RouteVideoUpload, controllers.UploadVideo)
	protected.GET(utils.RouteVideoList, controllers.GetVideos)
	protected.DELETE(utils.RouteVideoDelete, controllers.DeleteVideo)
	protected.PATCH(utils.RouteVideoUpdate, controllers.EditVideo)

	// Declaration routes
	protected.POST(utils.RouteDeclarationCreate, controllers.CreateDeclaration)
	protected.GET(utils.RouteDeclarationList, controllers.ListDeclarations)
	protected.PUT(utils.RouteDeclarationUpdate, controllers.UpdateDeclaration)
	protected.DELETE(utils.RouteDeclarationDelete, controllers.DeleteDeclarationForm)

	// Research Category routes
	researchCategoryRoutes := protected.Group("/research-categories")
	{
		researchCategoryRoutes.POST("", controllers.CreateResearchCategory)
		researchCategoryRoutes.GET("", controllers.ListCategories)
	}

	// Research routes
	researchRoutes := protected.Group("/research")
	{
		researchRoutes.POST("", controllers.CreateResearch)
		researchRoutes.GET("", controllers.ListResearches)
		researchRoutes.DELETE("/:id", controllers.DeleteResearch)
		researchRoutes.GET("/:id/document", controllers.GetResearchDocument)
		researchRoutes.PUT("/:id", controllers.UpdateResearch)
		researchRoutes.PATCH("/:id/publish", controllers.PublishResearch)
	}

	// E-Consent routes
	eConsentRoutes := protected.Group("/e-consent")
	{
		eConsentRoutes.POST("", controllers.CreateEConsentForm)
		eConsentRoutes.GET("", controllers.ListEConsentForms)
		eConsentRoutes.PUT("/:id", controllers.EditEConsentForm)
		eConsentRoutes.DELETE("/:id", controllers.DeleteEConsentForm)
		eConsentRoutes.PATCH("/:id/publish", controllers.PublishEConsentForm)
	}

	// Patient management routes
	protected.GET(utils.RoutePatientManagementList, controllers.ListPatientConsentAssignments)
	protected.GET(utils.RoutePatientConsentPDF, controllers.GeneratePatientConsentPDFHandler)
	protected.PATCH(utils.RoutePatientEConsentManageWithdrawl, controllers.NurseReviewWithdrawnConsent)

	// Static routes
	router.Static(utils.RouteStaticVideos, "./public/videos")
	router.Static(utils.RouteStaticResearchDoc, "./public/research_documents")

	// Handle 404
	router.NoRoute(controllers.NotFoundHandler)

	// router.POST("/generate-pdf", controllers.GenerateConsentPDFHandler)

	// Patient Routes
	protected.POST(utils.RoutePatientEconsent, controllers.CreatePatientEConsent)
	protected.POST(utils.RoutePatientEconsentRequestReSign, controllers.RequestResignPatientEConsent)
	protected.PATCH(utils.RoutePatientEConsentAdminSign, controllers.AdminSignPatientEConsent)
	protected.GET(utils.RoutePatientMyEConsents, controllers.GetPatientMyEConsents)
	protected.PATCH(utils.RoutePatientEConsentResign, controllers.PatientResignEConsent)
	protected.GET(utils.RoutePatientEconsentID, controllers.GetPatientEConsentFormDetails)
	protected.GET(utils.RouteAdminPatientEconsentID, controllers.GetPatientEConsentFormDetails)
	protected.GET(utils.RoutePatientEconsentAssigned, controllers.GetPatientAssignedEConsents)
	protected.PATCH(utils.RoutePatientEconsentWithdrawal, controllers.PatientWithdrawEConsent)
	protected.GET(utils.RoutePatientGenomicResults, controllers.GetPatientGenomicResults)

	//Patient Verification Routes
	protected.POST(utils.RoutePatientEconsentSendVerification, controllers.SendEConsentVerificationCode)
	protected.POST(utils.RoutePatientEconsentVerifyCode, controllers.VerifyEConsentVerificationCode)

	// Genomic Management routes
	protected.GET(utils.RouteGenomicList, controllers.ListConsentedPatients)
	protected.POST(utils.RouteGenomicResultUpload, controllers.UploadGenomicResult)
	protected.GET(utils.RouteGenomicResults, controllers.ListGenomicResults)
	protected.PUT(utils.RouteGenomicResultEdit, controllers.EditGenomicResult)
	protected.DELETE(utils.RouteGenomicResultDelete, controllers.DeleteGenomicResult)
	protected.PATCH(utils.RouteGenomicResultPublish, controllers.PublishGenomicResult)

	// Genomic Result File routes
	protected.GET(utils.RouteGenomicResultServe, controllers.ServeGenomicResultFile)
	protected.GET(utils.RouteGenomicResultDownload, controllers.DownloadGenomicResultFile)

	// consent form  routes

	protected.POST(utils.RouteCreateSession, controllers.CreateConsentSession)
	consent.GET(utils.RouteFormDetails, controllers.GetConsentSessionFormDetails)
	consent.GET(utils.RouteResearchDocument, controllers.GetConsentSessionResearchDocument)
	consent.POST(utils.RoutePatientEconsentSendVerification, controllers.SendConsentSessionVerificationCode)
	consent.POST(utils.RoutePatientEconsentVerifyCode, controllers.VerifyConsentSessionVerificationCode)
	consent.POST(utils.RouteSubmitConsentForm, controllers.CreateEConsentUsingSession)
}
