// middlewares/permission_middleware.go
package middlewares

import (
	"net/http"
	"regexp"
	"strings"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// RoutePermission defines which roles can access a specific route
type RoutePermission struct {
	Route  string   // Route path from utils.Routes
	Roles  []string // Allowed roles for this route
	Method string   // HTTP method (GET, POST, etc.)
}

// Define allowed roles
var AllowedRoles = []string{
	"super-admin",
	"admin",
	"user",
	"patient",
	"coordinator",
}

// RoutePermissions maps routes to their permitted roles
// Using the route constants from utils.Routes for consistency
var RoutePermissions = []RoutePermission{
	{
		Route:  "/api" + utils.RouteLogout,
		Roles:  []string{"super-admin", "admin", "user", "patient", "coordinator"},
		Method: http.MethodDelete,
	},
	{
		Route:  "/api" + utils.RouteResetPassword,
		Roles:  []string{"super-admin", "admin", "user", "patient", "coordinator"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteUpdateUser,
		Roles:  []string{"super-admin", "admin", "user", "patient", "coordinator"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteGetUserProfile,
		Roles:  []string{"super-admin", "admin", "user", "patient", "coordinator"},
		Method: http.MethodGet,
	},

	{
		Route:  "/api" + utils.RouteGetAdminPatientList,
		Roles:  []string{"admin", "super-admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteCreateAdminPatient,
		Roles:  []string{"admin", "super-admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteUpdateAdminPatientStatus,
		Roles:  []string{"admin", "super-admin"},
		Method: http.MethodPatch,
	},

	{
		Route:  "/api" + utils.RouteGetAdminUserList,
		Roles:  []string{"super-admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteCreateAdminUser,
		Roles:  []string{"super-admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteUpdateAdminUserProfile,
		Roles:  []string{"super-admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteUpdateAdminUserPassword,
		Roles:  []string{"super-admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteUpdateAdminUserStatus,
		Roles:  []string{"super-admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteDeleteAdminUser,
		Roles:  []string{"super-admin"},
		Method: http.MethodDelete,
	},
	{
		Route:  "/api" + utils.RouteKitInfo,
		Roles:  []string{"super-admin", "admin"},
		Method: "", // Empty means allow all methods
	},
	{
		Route:  "/api" + utils.RouteKitInfoID,
		Roles:  []string{"super-admin", "admin"},
		Method: "", // Empty means allow all methods
	},
	{
		Route:  "/api" + utils.RouteQuantitySummary,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteCustomer,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteOrder,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteAssignKit,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteChangeOrderStatus,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteviewCustomerWithOrders,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteNotification,
		Roles:  []string{"super-admin", "admin", "patient"},
		Method: "",
	},
	{
		Route:  "/api" + utils.RouteLatestNotification,
		Roles:  []string{"super-admin", "admin", "patient", "coordinator"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteMarkNotificationAsRead,
		Roles:  []string{"super-admin", "admin", "patient", "coordinator"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteMarkAllNotificationsAsRead,
		Roles:  []string{"super-admin", "admin", "patient", "coordinator"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteOrderCount,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteFetchedKitRegistrationList,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RoutePatientUpdateStatus,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RoutePatientUploadReport,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteDeleteNotification,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodDelete,
	},
	{
		Route:  "/api" + utils.RouteFetchedBarcodeDetails,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteFetchedLabsDetails,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteVideoUpload,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteVideoList,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteVideoDelete,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodDelete,
	},
	{
		Route:  "/api" + utils.RouteVideoUpdate,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteAdminDashboardStats,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteDeclarationCreate,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteDeclarationList,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteDeclarationUpdate,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPut,
	},
	{
		Route:  "/api" + utils.RouteDeclarationDelete,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodDelete,
	},

	// Research Category permissions
	{
		Route:  "/api" + utils.RouteResearchCategoryCreate,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteResearchCategoryList,
		Roles:  []string{"super-admin", "admin", "user"},
		Method: http.MethodGet,
	},

	// Research permissions
	{
		Route:  "/api" + utils.RouteResearchCreate,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteResearchList,
		Roles:  []string{"super-admin", "admin", "patient"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteResearchDelete,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodDelete,
	},
	{
		Route:  "/api/research/:id/document",
		Roles:  []string{"super-admin", "admin", "patient"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api/research/:id",
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPut,
	},
	{
		Route:  "/api" + utils.RouteResearchPublish,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api/e-consent",
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api/e-consent",
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteEConsentEdit,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPut,
	},
	{
		Route:  "/api/e-consent/:id",
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodDelete,
	},
	{
		Route:  "/api/admin/e-consent/dropdown-data",
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteEConsentPublish,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RoutePatientManagementList,
		Roles:  []string{"super-admin", "admin", "coordinator"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RoutePatientEConsent,
		Roles:  []string{"admin", "patient"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RoutePatientConsentPDF,
		Roles:  []string{"super-admin", "admin", "patient", "coordinator"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RoutePatientEConsentRequestResign,
		Roles:  []string{"super-admin", "admin", "coordinator"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RoutePatientEConsentAdminSign,
		Roles:  []string{"super-admin", "admin", "coordinator"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RoutePatientMyEConsents,
		Roles:  []string{"patient"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RoutePatientEConsentResign,
		Roles:  []string{"patient"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RoutePatientEConsentFormDetails,
		Roles:  []string{"super-admin", "admin", "patient"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RoutePatientEConsentAssignedList,
		Roles:  []string{"patient"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RoutePatientEconsentWithdrawal,
		Roles:  []string{"patient"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RoutePatientGenomicResults,
		Roles:  []string{"patient"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RoutePatientEConsentManageWithdrawl,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteGenomicList,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteGenomicResultUpload,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteGenomicResults,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteGenomicResultEdit,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPut,
	},
	{
		Route:  "/api" + utils.RouteGenomicResultDelete,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodDelete,
	},
	{
		Route:  "/api" + utils.RouteGenomicResultPublish,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RouteGenomicResultServe,
		Roles:  []string{"super-admin", "admin", "patient"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteGenomicResultDownload,
		Roles:  []string{"super-admin", "admin", "patient"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteUpdateAdminPatientDetails,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPatch,
	},
	{
		Route:  "/api" + utils.RoutePatientEconsentSendVerification,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RoutePatientEconsentVerifyCode,
		Roles:  []string{"super-admin", "admin"},
		Method: http.MethodPost,
	},
	{
		Route:  "/api" + utils.RouteAdminPatientEconsentID,
		Roles:  []string{"admin"},
		Method: http.MethodGet,
	},
	{
		Route:  "/api" + utils.RouteCreateSession,
		Roles:  []string{"admin"},
		Method: http.MethodPost,
	},
}

// CheckPermission checks if a user's role has permission for the given route and method
func CheckPermission(userRole, route, method string) bool {
	// Convert role to lowercase for consistent comparison
	userRole = strings.ToLower(userRole)

	// First check if the role is valid
	isValidRole := false
	for _, role := range AllowedRoles {
		if strings.ToLower(role) == userRole {
			isValidRole = true
			break
		}
	}
	if !isValidRole {
		return false
	}

	// Check permissions for the route
	for _, permission := range RoutePermissions {
		if permission.Route == "/api"+utils.RoutePatientConsentPDF {
			matched, _ := regexp.MatchString(`^/api/patient-management/\d+/pdf$`, route)
			if matched {
				if permission.Method != "" && permission.Method != method {
					continue
				}
				for _, allowedRole := range permission.Roles {
					if strings.ToLower(allowedRole) == userRole {
						return true
					}
				}
			}
			continue
		}
		// Convert route pattern to regex for matching
		routePattern := strings.Replace(permission.Route, ":id", "[^/]+", -1)
		routePattern = strings.Replace(routePattern, ":patientId", "[^/]+", -1)
		routePattern = strings.Replace(routePattern, ":filename", "[^/]+", -1)

		matched, _ := regexp.MatchString("^"+routePattern+"$", route)

		if matched {
			// If method is specified, it must match
			if permission.Method != "" && permission.Method != method {
				continue
			}

			// Check if user's role is allowed
			for _, allowedRole := range permission.Roles {
				if strings.ToLower(allowedRole) == userRole {
					return true
				}
			}
		}
	}

	return false
}

// CreatePermissionMiddleware creates a middleware that checks permissions
func CreatePermissionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by AuthMiddleware)
		user, ok := GetUserFromContext(c)
		if !ok {
			utils.JSONResponse(c, http.StatusUnauthorized, utils.MsgUserNotAuthenticated, nil)
			c.Abort()
			return
		}

		// Check if user has permission for this route
		hasPermission := CheckPermission(user.Role.Name, c.Request.URL.Path, c.Request.Method)
		if !hasPermission {
			utils.JSONResponse(c, http.StatusForbidden, utils.MsgAccessDeniedForOtherUser, nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
